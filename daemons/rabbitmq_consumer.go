package main

import (
	"encoding/json"
	"flag"
	"github.com/streadway/amqp"
	"log"
	"net/url"
	"os"
	"smarteragents/ellimango"
	"strconv"
	"strings"
	"time"
)

var env string
var oraUser string
var oraPass string
var oraSid string
var redisHost string
var redisDb int
var rabbitmqUser string
var rabbitmqHost string
var rabbitmqVhost string
var edgeEllimanApiUrl string
var ellimanApiUrl string

func init() {
	flag.StringVar(&env, "env", "", "Environment (local|staging|production).")
	flag.StringVar(&oraUser, "oraUser", "pde", "Oracle user.")
	flag.StringVar(&oraPass, "oraPass", "p", "Oracle password.")
	flag.StringVar(&oraSid, "oraSid", "de_dev", "Oracle sid.")
	// for redisHost use crestage.dtrleo.0001.use1.cache.amazonaws.com
	flag.StringVar(&redisHost, "redisHost", "", "Redis host.")
	// for redisDb use database 4 or 5 or 6 or 7 or 8
	flag.IntVar(&redisDb, "redisDb", 4, "Redis db.")
	flag.StringVar(&rabbitmqUser, "rabbitmqUser", "", "Rabbitmq user.")
	flag.StringVar(&rabbitmqHost, "rabbitmqHost", "", "Rabbitmq host.")
	flag.StringVar(&rabbitmqVhost, "rabbitmqVhost", "", "Rabbitmq vhost.")
	flag.StringVar(&edgeEllimanApiUrl, "edgeEllimanApiUrl", "", "Edge elliman api url for requests from elliman.")
	flag.StringVar(&ellimanApiUrl, "ellimanApiUrl", "", "Elliman api url.")
	// Parse command-line flags.
	flag.Parse()
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	if env == "" {
		log.Println(PLS_SP_ENV)
		os.Exit(1)
	}

	if env != "local" && env != "staging" && env != "production" {
		log.Printf("Wrong env `%s`.", env)
		os.Exit(1)
	}

	if redisHost == "" {
		log.Println(PLS_SP_REDIS_HOST)
		os.Exit(1)
	}

	if rabbitmqUser == "" {
		log.Println(PLS_SP_RABBITMQ_USR)
		os.Exit(1)
	}

	if rabbitmqHost == "" {
		log.Println(PLS_SP_RABBITMQ_HOST)
		os.Exit(1)
	}

	if rabbitmqVhost == "" {
		log.Println(PLS_SP_RABBITMQ_VHOST)
		os.Exit(1)
	}

	if edgeEllimanApiUrl == "" {
		log.Println(PLS_SP_EDGE_ELLIMAN_URL)
		os.Exit(1)
	}

	if ellimanApiUrl == "" {
		log.Println(PLS_SP_ELLIMAN_URL)
		os.Exit(1)
	}

	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}
	agent := ellimango.Agent{Oracle: oracle,
		Redis: redis,
	}
	userListing := ellimango.UserListing{Oracle: oracle,
		Redis: redis,
	}
	folder := ellimango.Folder{Oracle: oracle,
		Redis: redis,
	}

	conn, ch, err := rabbitmq.Connect()
	failOnError(err, "Failed to connect to RabbitMQ")
	defer rabbitmq.Close(conn, ch)

	// no connection error, no channel opening error
	// start consume
	q, err := ch.QueueDeclare(
		RABBITMQ_TASKS_QUEUE+env, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var task ellimango.RabbitmqTask
			err = json.Unmarshal(d.Body, &task)
			if err != nil {
				log.Println("Json unmarshall error", err)
			} else {
				// create switch from task.SyncAction
				// depending on action do different actions
				switch task.SyncAction {
				case MAP_PDE_USER_AGENT:
					// send user agent to edge elliman
					go sendUserAgentToEdgeElliman(&task, &user, &agent, ch, &q, &d)

				case MOD_PDE_USER:
					// send user to edge elliman
					go sendUserToEdgeElliman(&task, &user, &agent, ch, &q, &d)

				case ADD_SVD_APT:
					// send user saved listing to edge elliman
					go sendSavedApartmentToEdgeElliman(&task, &folder, ch, &q, &d)

				case REDIS_UPD_USR_LSTS:
					// get user listings from oracle and update redis
					go redisUpdateUserListings(&task, &userListing, ch, &q, &d)

				case REDIS_UPD_USR:
					// get user from oracle and update redis
					go redisUpdateUser(&task, &user, ch, &q, &d)

				case MEMB_VER_EMAIL:
					// make a call to elliman api to send membership verification email to user from elliman
					go membershipVerificationEmail(&task, &user, ch, &q, &d)

				case DEL_SVD_APT:
					// send delete user saved listing to edge elliman
					go sendDeleteSavedApartmentToEdgeElliman(&task, &folder, ch, &q, &d)

				}
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
	// end consume
}

// helper - to get oracle, redis, rabbitmq helpers
func getOracleRedisRabbitmqHelpers() (ellimango.Oracle, ellimango.Redis, ellimango.Rabbitmq) {
	oracle := ellimango.Oracle{Env: env,
		OraUser: oraUser,
		OraPass: oraPass,
		OraSid:  oraSid,
	}
	redis := ellimango.Redis{Env: env,
		RedisHost: redisHost,
		RedisDb:   redisDb,
	}
	rabbitmq := ellimango.Rabbitmq{Env: env,
		RabbitmqUser:  rabbitmqUser,
		RabbitmqHost:  rabbitmqHost,
		RabbitmqVhost: rabbitmqVhost,
	}

	return oracle, redis, rabbitmq
}

// sync user agent to edge elliman
func sendUserAgentToEdgeElliman(message *ellimango.RabbitmqTask, user *ellimango.User, agent *ellimango.Agent, ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	helper := ellimango.Helper{Env: env}
	var userWithP ellimango.UserWithP
	var foAgtInfo ellimango.FoAgtInfo
	var err error
	var processingFailed bool

	helper.Debug("sendUserAgentToEdgeElliman")

	// get user info
	if message.DeUserId != "" {
		userWithP, err = user.GetByUserId(message.DeUserId)
		if err != nil {
			log.Println("sendUserAgentToEdgeElliman user.GetByUserId error", err)
		}
	}

	// get agent info by user id
	if message.DeUserId != "" {
		foAgtInfo, err = agent.GetByUserId(message.DeUserId)
		if err != nil {
			log.Println("sendUserAgentToEdgeElliman agent.GetByUserId error", err)
		}
	}

	// send data to edge elliman
	response, err := postUserAgentToEdge(&userWithP, &foAgtInfo)
	helper.Debug(err)
	helper.Debug(string(response))

	if `{"status":"success","message":null}` != string(response) {
		helper.Debug("Processing Failed")
		processingFailed = true
	} else {
		helper.Debug("Processing Success")
	}

	// Ack message
	d.Ack(false)
	// if processing failed, put message back into queue
	if processingFailed == true {
		republish(ch, q, d)
	}
}

// sync user to edge elliman
func sendUserToEdgeElliman(message *ellimango.RabbitmqTask, user *ellimango.User, agent *ellimango.Agent, ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	helper := ellimango.Helper{Env: env}
	var userWithP ellimango.UserWithP
	var foAgtInfo ellimango.FoAgtInfo
	var err error
	var processingFailed bool

	helper.Debug("sendUserToEdgeElliman")

	// get user info
	if message.DeUserId != "" {
		userWithP, err = user.GetByUserId(message.DeUserId)
		if err != nil {
			log.Println("sendUserToEdgeElliman user.GetByUserId error", err)
		}
	}

	// get agent info
	if message.DeUserId != "" {
		foAgtInfo, err = agent.GetByUserId(message.DeUserId)
		if err != nil {
			log.Println("sendUserToEdgeElliman agent.GetByAgentId error", err)
		}
	}

	formData := url.Values{}
	formData.Set("login", "pde_user9000")
	formData.Set("password", "usfafgwe")
	formData.Set("action", MOD_PDE_USER)

	// user found
	if userWithP.DeUserId != "" {
		go postToSendMembershipVerificationEmail(userWithP.DeUserId)
		formData.Set("pde_user_id", userWithP.DeUserId)
		formData.Set("username", userWithP.Email)
		formData.Set("user_password", userWithP.P)
		formData.Set("first_name", userWithP.Firstname)
		formData.Set("last_name", userWithP.Lastname)
		formData.Set("email", userWithP.Email)
		formData.Set("day_phone", userWithP.Phone)
		formData.Set("ip", userWithP.OriginIp)
	} else {
		log.Println("user not found, strange", userWithP, "string(d.Body)=", string(d.Body), "message=", message)
	}

	// user has agent
	if foAgtInfo.Id != "" {
		formData.Set("agent_name", foAgtInfo.Firstname+" "+foAgtInfo.Lastname)
		formData.Set("agent_email", foAgtInfo.Email)
		formData.Set("agent_id", foAgtInfo.Id)
		// user does not have agent
	} else {
		formData.Set("cr_actionable", "1")
	}

	// send data to edge elliman
	response, err := helper.Post(edgeEllimanApiUrl+"PdeUser", formData)
	helper.Debug(formData)
	helper.Debug(err)
	helper.Debug(string(response))

	if `{"status":"success","message":null}` != string(response) {
		helper.Debug("Processing Failed")
		processingFailed = true
	} else {
		helper.Debug("Processing Success")
		// send user agent info to edge elliman
		if foAgtInfo.Id != "" {
			response, err := postUserAgentToEdge(&userWithP, &foAgtInfo)
			helper.Debug(err)
			helper.Debug(string(response))
		}
	}

	// Ack message
	d.Ack(false)
	// if processing failed, put message back into queue
	if processingFailed == true {
		republish(ch, q, d)
	}
}

// sync saved listing to edge elliman
func sendSavedApartmentToEdgeElliman(message *ellimango.RabbitmqTask, folder *ellimango.Folder, ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	// select id, pde_listing_id, pde_user_id, CR_LISTING_ID, cr_usr_id, url, folder, price, trunc(date_created), sale_or_rent, hood,address, bedrooms, agency, rebny_id, agent_usr_id, status_id, THIRD_PARTY_ID, REGION_ID, NEIGHBORHOOD_ID, LONGITUDE, LATITUDE, FOLDER_ACCESS_TYPE, area, doorman, BUILDING_TYPE_ID from cr_pde_saved_apartments where id>=539620 and pde_user_id=99885;
	helper := ellimango.Helper{Env: env}
	var solrListing ellimango.SolrListing
	var folderName string
	var errorMessage string
	var processingFailed bool

	helper.Debug("sendSavedApartmentToEdgeElliman")

	// Listing id present
	if message.ListingId != "" {
		solrListing, _ = helper.GetSolrListingDataForAddSavedApartment(message.ListingId)
		//helper.Debug("solrLIsting", solrListing)
	} else {
		log.Println("ListingId was empty in rabbitmq message")
	}

	// Folder id present
	if message.FolderId != "" {
		folderName, errorMessage = folder.GetListingFolderNameByFolderId(message.FolderId)
		if errorMessage != "" {
			log.Println("Daemon sendSavedApartmentToEdgeElliman folder.GetListingFolderNameByFolderId errorMessage=", errorMessage)
		}
		// Folder id missing
	} else {
		log.Println("FolderId was empty in rabbitmq message")
	}

	// folder name and listing data are present
	// send data to edge elliman
	if folderName != "" && solrListing.Tpid != "" {
		var listingType string
		if solrListing.TransTypeId == 2 {
			listingType = "Sale"
		} else if solrListing.TransTypeId == 1 {
			listingType = "Rental"
		}
		price := strings.Replace(solrListing.CurrentPrice, ".00,USD", "", 1)
		var bedrooms string
		if solrListing.NumBedrooms == 0 {
			if strings.Contains(solrListing.DisplayAttribute, "Loft") {
				bedrooms = "Loft"
			} else {
				bedrooms = "Studio"
			}
		} else if solrListing.NumBedrooms == 1 {
			bedrooms = "1 Bedroom"
		} else {
			bedrooms = strconv.FormatFloat(float64(solrListing.NumBedrooms), 'f', -1, 32) + " Bedrooms"
		}

		formData := url.Values{}
		formData.Set("login", "pde_user9000")
		formData.Set("password", "usfafgwe")
		formData.Set("pde_user_id", message.DeUserId)
		formData.Set("third_party_id", solrListing.Tpid)
		formData.Set("pde_listing_id", message.ListingId)
		formData.Set("agency", solrListing.AgencyName)
		formData.Set("listing_type", listingType)
		formData.Set("hood", solrListing.NeighborhoodName)
		formData.Set("folder", folderName)
		// no description
		formData.Set("url", solrListing.Url)
		formData.Set("price", price)
		formData.Set("address", solrListing.DisplayName)
		formData.Set("bedrooms", bedrooms)
		formData.Set("date_created", strconv.FormatInt(time.Now().Unix(), 10))
		// no ip
		formData.Set("neighborhood_id", solrListing.NeighborhoodId)
		formData.Set("region_id", solrListing.LclRegionId)
		formData.Set("pde_building_id", strconv.FormatUint(solrListing.BuildingId, 10))
		formData.Set("longitude", solrListing.Longitude)
		formData.Set("latitude", solrListing.Latitude)
		formData.Set("area", strconv.FormatFloat(float64(solrListing.Area), 'f', 5, 32))
		formData.Set("full_time_doorman", strconv.FormatUint(uint64(solrListing.FullTimeDoorman), 10))
		formData.Set("part_time_doorman", strconv.FormatUint(uint64(solrListing.PartTimeDoorman), 10))
		formData.Set("building_type_id", strconv.FormatUint(uint64(solrListing.BldgTypeId), 10))
		formData.Set("affiliate_id", strconv.FormatUint(uint64(solrListing.AffiliateId), 10))
		formData.Set("action", "add_saved_apartment")

		// send data to edge elliman
		response, err := helper.Post(edgeEllimanApiUrl+"AddSavedApartment", formData)
		helper.Debug(formData)
		helper.Debug(err)
		helper.Debug(string(response))

		if `{"status":"success","message":null}` != string(response) {
			helper.Debug("Processing Failed")
			processingFailed = true
		} else {
			helper.Debug("Processing Success")
		}
		// some data missing, do not send to edge elliman
	} else {
		if folderName == "" {
			log.Println("folderName is empty, perhaps folder was deleted")
		}
		if solrListing.Tpid == "" {
			log.Println("solrListing tpid is empty, listing is missing from solr")
		}
		// processingFailed = true
	}

	// Ack message
	d.Ack(false)
	// if processing failed, put back into queue
	if processingFailed == true {
		republish(ch, q, d)
	}
}

// sync delete saved listing to edge elliman
func sendDeleteSavedApartmentToEdgeElliman(message *ellimango.RabbitmqTask, folder *ellimango.Folder, ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	// select id, pde_listing_id, pde_user_id, CR_LISTING_ID, cr_usr_id, url, folder, price, trunc(date_created), sale_or_rent, hood,address, bedrooms, agency, rebny_id, agent_usr_id, status_id, THIRD_PARTY_ID, REGION_ID, NEIGHBORHOOD_ID, LONGITUDE, LATITUDE, FOLDER_ACCESS_TYPE, area, doorman, BUILDING_TYPE_ID from cr_pde_saved_apartments where id>=539620 and pde_user_id=99885;
	helper := ellimango.Helper{Env: env}
	var folderName string
	var errorMessage string
	var processingFailed bool

	helper.Debug("sendDeleteSavedApartmentToEdgeElliman")

	// Listing id not present
	if message.ListingId == "" {
		log.Println("ListingId was empty in rabbitmq message")
	}

	// Folder id present
	if message.FolderId != "" {
		folderName, errorMessage = folder.GetListingFolderNameByFolderId(message.FolderId)
		if errorMessage != "" {
			log.Println("Daemon sendDeleteSavedApartmentToEdgeElliman folder.GetListingFolderNameByFolderId errorMessage=", errorMessage)
		}
		// Folder id missing
	} else {
		log.Println("FolderId was empty in rabbitmq message")
	}

	// folder name is present
	// send data to edge elliman
	if folderName != "" {
		formData := url.Values{}
		formData.Set("login", "pde_user9000")
		formData.Set("password", "usfafgwe")
		formData.Set("pde_user_id", message.DeUserId)
		formData.Set("pde_listing_id", message.ListingId)
		formData.Set("folder", folderName)
		formData.Set("action", "delete_saved_apartment")

		// send data to edge elliman
		response, err := helper.Post(edgeEllimanApiUrl+"DeleteSavedApartment", formData)
		helper.Debug(formData)
		helper.Debug(err)
		helper.Debug(string(response))

		if `{"status":"success","message":null}` != string(response) {
			subStr := "pde_user_id not found in cr_pde_usr_map"
			subStr2 := "cr_pde_saved_apartments not found"
			if strings.Contains(string(response), subStr) {
				helper.Debug("Processing Failed, but will not republish. pde_user_id not found")
				processingFailed = false
			} else if strings.Contains(string(response), subStr2) {
				helper.Debug("Processing Failed, but will not republish. cr_pde_saved_apartments not found")
				processingFailed = false
			} else {
				helper.Debug("Processing Failed")
				processingFailed = true
			}
		} else {
			helper.Debug("Processing Success")
		}
		// some data missing, do not send to edge elliman
	} else {
		if folderName == "" {
			log.Println("folderName is empty, perhaps folder was deleted")
		}
		// processingFailed = true
	}

	// Ack message
	d.Ack(false)
	// if processing failed, put back into queue
	if processingFailed == true {
		republish(ch, q, d)
	}
}

// get user listings from oracle, update redis
func redisUpdateUserListings(message *ellimango.RabbitmqTask, userListing *ellimango.UserListing, ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	helper := ellimango.Helper{Env: env}
	var processingFailed bool

	helper.Debug("redisUpdateUserListings")

	if message.DeUserId != "" {
		errorMessage := userListing.RedisUpdate(message.DeUserId)
		if errorMessage != "" {
			helper.Debug("Processing Failed")
			processingFailed = true
		} else {
			helper.Debug("Processing Success")
		}
	} else {
		log.Println("DeUserId was missing in message", message)
	}

	// Ack message
	d.Ack(false)
	// if processing failed, put back into queue
	if processingFailed == true {
		republish(ch, q, d)
	}
}

// get user from oracle, update redis
func redisUpdateUser(message *ellimango.RabbitmqTask, user *ellimango.User, ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	helper := ellimango.Helper{Env: env}
	var processingFailed bool

	helper.Debug("redisUpdateUser")

	if message.DeUserId != "" {
		errorMessage := user.RedisUpdate(message.DeUserId)
		if errorMessage != "" {
			helper.Debug("Processing Failed")
			processingFailed = true
		} else {
			helper.Debug("Processing Success")
		}
	} else {
		log.Println("DeUserId was missing in message", message)
	}

	// Ack message
	d.Ack(false)
	// if processing failed, put back into queue
	if processingFailed == true {
		republish(ch, q, d)
	}
}

// make a call to elliman api to send email to user
func membershipVerificationEmail(message *ellimango.RabbitmqTask, user *ellimango.User, ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	helper := ellimango.Helper{Env: env}
	var processingFailed bool

	helper.Debug("membershipVerificationEmail")

	if message.DeUserId != "" {
		response, err := postToSendMembershipVerificationEmail(message.DeUserId)
		if err != nil {
			log.Println("membershipVerificationEmail postToSendMembershipVerificationEmail err", err)
			processingFailed = true
		} else {
			if `{"status":"success","message":"Email sent"}` != string(response) {
				helper.Debug("Processing Failed")
				processingFailed = true
			} else {
				helper.Debug("Processing Success")
			}
		}
	} else {
		log.Println("DeUserId was missing in message", message)
	}

	// Ack message
	d.Ack(false)
	// if processing failed, put back into queue
	if processingFailed == true {
		republish(ch, q, d)
	}
}

// helper - to publish previously consumed message back to rabbitmq
func republish(ch *amqp.Channel, q *amqp.Queue, d *amqp.Delivery) {
	helper := ellimango.Helper{Env: env}
	helper.Debug("Republishing")
	err := ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         d.Body,
		})
	//log.Printf("[x] Republished %s", d.Body)
	if err != nil {
		log.Println("Rabbitmq republish error", err)
		// go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Rabbitmq publish error %v", err), "Rabbitmq publish error")
	}
}

// helper - post data to elliman api to send membership verification email
func postToSendMembershipVerificationEmail(deUserId string) ([]byte, error) {
	helper := ellimango.Helper{Env: env}
	helper.Debug("postToSendMembershipVerificationEmail")
	formData := url.Values{}
	formData.Set("hk", "n#EQwdCUG@%^Ld8S")
	formData.Set("action", "send-membership-verification-email")
	formData.Set("rand", strconv.FormatInt(time.Now().UnixNano(), 10))
	formData.Set("user_id", deUserId)

	// send data to elliman
	response, err := helper.Post(ellimanApiUrl, formData)
	helper.Debug(formData)
	helper.Debug(err)
	helper.Debug(string(response))

	return response, err
}

// helper - post user agent data to edge elliman
func postUserAgentToEdge(userWithP *ellimango.UserWithP, foAgtInfo *ellimango.FoAgtInfo) ([]byte, error) {
	helper := ellimango.Helper{Env: env}

	formData := url.Values{}
	formData.Set("login", "pde_user9000")
	formData.Set("password", "usfafgwe")
	formData.Set("action", MAP_PDE_USER_AGENT)

	// user found
	if userWithP.DeUserId != "" {
		formData.Set("pde_user_id", userWithP.DeUserId)
		formData.Set("user_username", userWithP.Email)
		formData.Set("user_email", userWithP.Email)
		formData.Set("user_first_name", userWithP.Firstname)
		formData.Set("user_last_name", userWithP.Lastname)
		formData.Set("user_day_phone", userWithP.Phone)
		formData.Set("user_password", userWithP.P)

	}

	// agent found
	if foAgtInfo.Id != "" {
		formData.Set("pde_agent_id", foAgtInfo.Id)
		formData.Set("agent_id", foAgtInfo.Id)
		formData.Set("agent_email", foAgtInfo.Email)
		formData.Set("agent_third_party_id", foAgtInfo.Tpid)
		formData.Set("agent_first_name", foAgtInfo.Firstname)
		formData.Set("agent_last_name", foAgtInfo.Lastname)
		formData.Set("agent_status", foAgtInfo.Status)
		formData.Set("agent_mobile_number", foAgtInfo.Mobile)
		formData.Set("agent_phone_number", foAgtInfo.Phone)
		formData.Set("agent_fax_number", foAgtInfo.Fax)
		formData.Set("agent_photo_url", foAgtInfo.PhotoUrl)
		formData.Set("agent_offices", foAgtInfo.Offices)
	}

	response, err := helper.Post(edgeEllimanApiUrl+"MapPdeUserAgent", formData)

	return response, err
}

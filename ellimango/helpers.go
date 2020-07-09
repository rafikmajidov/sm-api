package ellimango

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

//lst_ref_affiliate
const (
	LstAffNewYorkCity         = "1"
	LstAffLongIsland          = "2"
	LstAffHamptons            = "3"
	LstAffMLSNonIdx           = "4"
	LstAffOutsideNYCLI        = "5"
	LstAffWestchester         = "6"
	LstAffFlorida             = "7"
	LstAffCalifornia          = "8"
	LstAffConnecticut         = "9"
	LstAffAspen               = "10"
	LstAffiliateMassachusetts = "11"
)

const (
	EllimanAffNewYorkCity   = "0"
	EllimanAffLongIsland    = "1"
	EllimanAffHamptons      = "2"
	EllimanAffMLSNonIdx     = "3"
	EllimanAffOutsideNYCLI  = "4"
	EllimanAffWestchester   = "6"
	EllimanAffFlorida       = "7"
	EllimanAffCalifornia    = "8"
	EllimanAffConnecticut   = "5"
	EllimanAffAspen         = "9"
	EllimanAffMassachusetts = "10"
)

type Helper struct {
	Env string
}

func (h *Helper) Debug(messages ...interface{}) {
	if h.Env == "local" || h.Env == "staging" || h.Env == "production" {
		// var allMessages interface{}
		for i := range messages {
			log.Println(messages[i])
		}
	}
}
//migration/api/core.php: mig_convert_db_affiliate_id_to_elliman_feed_affiliate_id
func (h *Helper) ConvertDbAffiliateIdToEllimanAffiliateId(dbAffiliateId string) string {
	var ellimanAffiliateId string
	switch dbAffiliateId {
	case LstAffNewYorkCity:
		ellimanAffiliateId = EllimanAffNewYorkCity

	case LstAffLongIsland:
		ellimanAffiliateId = EllimanAffLongIsland

	case LstAffHamptons:
		ellimanAffiliateId = EllimanAffHamptons

	case LstAffMLSNonIdx:
		ellimanAffiliateId = EllimanAffMLSNonIdx

	case LstAffOutsideNYCLI:
		ellimanAffiliateId = EllimanAffOutsideNYCLI

	case LstAffWestchester:
		ellimanAffiliateId = EllimanAffWestchester

	case LstAffFlorida:
		ellimanAffiliateId = EllimanAffFlorida

	case LstAffCalifornia:
		ellimanAffiliateId = EllimanAffCalifornia

	case LstAffConnecticut:
		ellimanAffiliateId = EllimanAffConnecticut

	case LstAffAspen:
		ellimanAffiliateId = EllimanAffAspen

	case LstAffiliateMassachusetts:
		ellimanAffiliateId = EllimanAffMassachusetts
	}
	return ellimanAffiliateId
}

//migration/api/core.php: _mig_convert_pde_affiliate_id_to_affiliate_id
func (h *Helper) ConvertEllimanAffiliateIdToDbAffiliateId(affiliateId string) string {
	var dbAffiliateId string
	switch affiliateId {
	case EllimanAffNewYorkCity:
		dbAffiliateId = LstAffNewYorkCity

	case EllimanAffLongIsland:
		dbAffiliateId = LstAffLongIsland

	case EllimanAffHamptons:
		dbAffiliateId = LstAffHamptons

	case EllimanAffMLSNonIdx:
		dbAffiliateId = LstAffMLSNonIdx

	case EllimanAffOutsideNYCLI:
		dbAffiliateId = LstAffOutsideNYCLI

	case EllimanAffWestchester:
		dbAffiliateId = LstAffWestchester

	case EllimanAffFlorida:
		dbAffiliateId = LstAffFlorida

	case EllimanAffCalifornia:
		dbAffiliateId = LstAffCalifornia

	case EllimanAffConnecticut:
		dbAffiliateId = LstAffConnecticut

	case EllimanAffAspen:
		dbAffiliateId = LstAffAspen

	case EllimanAffMassachusetts:
		dbAffiliateId = LstAffiliateMassachusetts

	default:
		dbAffiliateId = ""
	}
	return dbAffiliateId
}

func (h *Helper) SendEmail(toName string, toEmail string, body string, subject string) {
	// Set up authentication information.
	// https://gist.github.com/andelf/5004821
	auth := smtp.PlainAuth("", "majidov.rafik.reol@gmail.com", "Rf1k99#$*(d77", "smtp.gmail.com")

	from := mail.Address{"Elliman go api server", "majidov.rafik.reol@gmail.com"}
	to := mail.Address{toName, toEmail}

	header := make(map[string]string)
	header["From"] = from.String()
	header["To"] = to.String()
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))

	err := smtp.SendMail("smtp.gmail.com:587", auth, from.Address, []string{to.Address}, []byte(message))
	if err != nil {
		log.Println("Goemail send error", err)
	} else {
		log.Println("Goemail send ok")
	}
}

func (h *Helper) Post(url string, formData url.Values) ([]byte, error) {
	var responseBody []byte

	client := &http.Client{}
	requestBody := strings.NewReader(formData.Encode())
	req, err := http.NewRequest("POST", url, requestBody)
	if err != nil {
		// handle err
		log.Println("Helper Post http.NewRequest error", err)
	} else {
		// req.SetBasicAuth("cityrealty", "275seventh")
		req.SetBasicAuth("elliman", "1911")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err2 := client.Do(req)
		if err2 != nil {
			// handle err
			log.Println("Helper Post http.DefaultClient.Do error", err2)
			err = err2
		} else {
			responseBody, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("Helper Post ioutil.ReadAll error", err)
			}
			defer resp.Body.Close()
		}
	}
	return responseBody, err
	/*
		resp, err := http.PostForm(url, formData)
			if err != nil {
				log.Println("postform error", err)
			} else {

				text, err := ioutil.ReadAll(resp.Body)

				defer resp.Body.Close()

				log.Printf("%s, %v, %d\n", text, err, resp.StatusCode)
			}
	*/
}

func (h *Helper) GetSolrListingDataForAddSavedApartment(listingId string) (SolrListing, error) {
	var solrListingResponse SolrListingResponse
	var solrListing SolrListing
	var solrUrl string
	if h.Env == "local" || h.Env == "staging" {
		solrUrl = "http://192.168.50.224:8983/solr/elliman_staging/select"
	} else if h.Env == "production" {
		solrUrl = "http://solr.reol.com/solr/pde_frontend/select"
	}
	getUrl := solrUrl + "?q=item_type:Listing+AND+item_id:" + listingId + "&wt=json&fl=third_party_id,building_type_id,transaction_type_id,current_price,region_id,neighborhood_id,num_bedrooms,area,status_id,agency_name,lcl_region_id,neighborhood_name,longitude,latitude,display_name,url,full_time_doorman,part_time_doorman,building_id,affiliate_id,display_attribute"
	resp, err := http.Get(getUrl)
	if err != nil {
		// handle error
		log.Println("Helper GetSolrListingDataForAddSavedApartment http.Get error", err)
	} else {
		responseBody, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			log.Println("Helper GetSolrListingDataForAddSavedApartment ioutil.ReadAll error", err2)
			err = err2
		} else {
			err = json.Unmarshal(responseBody, &solrListingResponse)
			if err == nil {
				if solrListingResponse.ResponseInner.NumFound == 1 {
					solrListing = solrListingResponse.ResponseInner.Docs[0]
				}
			} else {
				log.Println("Helper GetSolrListingDataForAddSavedApartment Unmarshal error", err)
			}
		}
		defer resp.Body.Close()
	}

	return solrListing, err
}

func (h *Helper) GetRabbitmqStatus(rabbitmq Rabbitmq) (RabbitmqStatus, error) {
	var rabbitmqStatus RabbitmqStatus
	var responseBody []byte
	var url = "http://" + rabbitmq.RabbitmqHost + ":15672/api/queues/" + rabbitmq.RabbitmqVhost + "/" + RABBITMQ_TASKS_QUEUE + h.Env

	client := &http.Client{Timeout: time.Duration(3) * time.Second}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		// handle err
		log.Println("GetRabbitmqStatus http.NewRequest GET error", err)
	} else {
		req.SetBasicAuth("admin", "R@f1k99%d77")
		req.Header.Set("Content-Type", "application/json")
		resp, err2 := client.Do(req)

		if err2 != nil {
			// handle err
			log.Println("GetRabbitmqStatus http.DefaultClient.Do error", err2)
			err = err2
		} else {
			responseBody, err = ioutil.ReadAll(resp.Body)

			if err != nil {
				log.Println("GetRabbitmqStatus ioutil.ReadAll error", err)
			} else {
				err = json.Unmarshal(responseBody, &rabbitmqStatus)
				if rabbitmqStatus.Name == "" || rabbitmqStatus.Node == "" || rabbitmqStatus.Memory == 0 {
					err = errors.New("Can not get rabbitmq status")
				}
			}
			defer resp.Body.Close()
		}
	}

	return rabbitmqStatus, err
}

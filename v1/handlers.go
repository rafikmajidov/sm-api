package main

import (
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"
	_ "reflect"
	"smarteragents/ellimango"
	"strconv"
	"strings"
	"time"
)

// Middleware is a function invoked by the routing layer before final request handler is
type Middleware func(fasthttp.RequestHandler) fasthttp.RequestHandler

// Handle accepts the request handler and variadic middleware functions
// It ranges over the middleware functions and invokes them before the final request handler
func Handle(h fasthttp.RequestHandler, mw ...Middleware) fasthttp.RequestHandler {
	for i := range mw {
		h = mw[i](h)
	}

	return func(ctx *fasthttp.RequestCtx) {
		h(ctx)
	}
}

// middleware - check that header has Authtoken,
// which is equal to requiredAuthtoken flag
func Auth(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		// Get the Authtoken header
		authtoken := string(ctx.Request.Header.Peek("Authtoken"))
		// Display error message
		if authtoken != requiredAuthtoken {
			outputResponse(ctx, 401, ellimango.Response{Reason: PLS_TRY_LATER})

			// Otherwise delegate request to the given handle
		} else {
			h(ctx)
		}
	})
}

// middleware - grab token if passed from url
// and inject into incoming header (for services that cant support)
func SetAuthtokenFromUrl(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		// get token from url
		sAuthtoken := ctx.UserValue("authtoken").(string)
		// token from url not empty, inject it into header
		if sAuthtoken != "" {
			ctx.Request.Header.Set("Authtoken", sAuthtoken)
		}
		h(ctx)
	})
}

// middleware - validate that userId is numeric
func ValidateUserId(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		sUserId := ctx.UserValue("userId").(string)
		iUserId, err := strconv.ParseUint(sUserId, 10, 64)

		// userId is numeric
		if iUserId > 0 && err == nil {
			h(ctx)
			// userId is not numeric
		} else {
			outputResponse(ctx, 400, ellimango.Response{Reason: "User(" + sUserId + ") is invalid"})
		}
	})
}

// middleware - validate that email and password are not empty
func ValidateLogin(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		email := strings.ToLower(getFormValue(ctx, "email"))
		password := getFormValue(ctx, "password")
		missingFields := make([]string, 0)

		if email == "" {
			missingFields = append(missingFields, "email")
		}

		if password == "" {
			missingFields = append(missingFields, "password")
		}

		if len(missingFields) > 0 {
			outputResponse(ctx, 400, ellimango.Response{Reason: PLS_FILL_FIELDS + strings.Join(missingFields, ",")})
			// no missing info
		} else {
			ctx.SetUserValue("email", email)
			ctx.SetUserValue("password", password)
			h(ctx)
		}
	})
}

// middleware - validate inputs when create/update user
func ValidateCreateUpdate(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		firstname := strings.ToLower(getFormValue(ctx, "firstname"))
		lastname := strings.ToLower(getFormValue(ctx, "lastname"))
		email := strings.ToLower(getFormValue(ctx, "email"))
		phone := getFormValue(ctx, "phone")
		password := getFormValue(ctx, "password")
		saAppCode := getFormValue(ctx, "sa_app_code")
		deAgentId := strings.ToUpper(getFormValue(ctx, "de_agent_id"))
		fbId := getFormValue(ctx, "fb_id")
		originIp := getFormValue(ctx, "origin_ip")
		missingFields := make([]string, 0)
		invalidFields := make([]string, 0)

		// validation
		if firstname == "" {
			missingFields = append(missingFields, "firstname")
		}

		if lastname == "" {
			missingFields = append(missingFields, "lastname")
		}

		if email == "" {
			missingFields = append(missingFields, "email")
		}

		if password == "" {
			missingFields = append(missingFields, "password")
		} else if len(password) < 4 {
			invalidFields = append(invalidFields, "password(must have at least 4 characters)")
		}

		if saAppCode == "" {
			missingFields = append(missingFields, "sa_app_code")
		}

		// empty fields
		if len(missingFields) > 0 {
			outputResponse(ctx, 400, ellimango.Response{Reason: PLS_FILL_FIELDS + strings.Join(missingFields, ",")})
			// invalid fields
		} else if len(invalidFields) > 0 {
			outputResponse(ctx, 400, ellimango.Response{Reason: INVALID_PASSWORD})
			// no missing info
		} else {
			ctx.SetUserValue("firstname", firstname)
			ctx.SetUserValue("lastname", lastname)
			ctx.SetUserValue("email", email)
			ctx.SetUserValue("phone", phone)
			ctx.SetUserValue("password", password)
			ctx.SetUserValue("saAppCode", saAppCode)
			ctx.SetUserValue("deAgentId", deAgentId)
			ctx.SetUserValue("fbId", fbId)
			ctx.SetUserValue("originIp", originIp)
			h(ctx)
		}
	})
}

// middleware - validate that agent with such and deAgentId exists
func ValidateDeAgentId(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		var agentId uint64
		deAgentId := ctx.UserValue("deAgentId").(string)

		// deAgentId not empty, try to find agent id by third party id
		if deAgentId != "" {
			oracle, redis, _ := getOracleRedisRabbitmqHelpers()
			agent := ellimango.Agent{Oracle: oracle,
				Redis: redis,
			}

			agentId, err := agent.GetAgentIdByThirdPartyId(deAgentId)
			// error
			if err != nil {
				outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
				// no error
			} else {
				// agent not found
				if agentId == 0 {
					outputResponse(ctx, 404, ellimango.Response{Reason: "Agent(" + deAgentId + ") does not exist"})
					// agent found
				} else {
					ctx.SetUserValue("agentId", agentId)
					h(ctx)
				}
			}
			// deAgentId empty, can continue
		} else {
			ctx.SetUserValue("agentId", agentId)
			h(ctx)
		}
	})
}

// middleware - validate that user with such email exists or does not exist
func ValidateEmailForCreate(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		email := ctx.UserValue("email").(string)

		oracle, redis, _ := getOracleRedisRabbitmqHelpers()
		user := ellimango.User{Oracle: oracle,
			Redis: redis,
		}
		userId, err := user.GetUserIdByEmail(email)
		// error
		if err != nil {
			outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
			// no error
		} else {
			// user found
			if userId != "" {
				outputResponse(ctx, 400, ellimango.Response{Reason: EMAIL_EXISTS})
				// user not found
			} else {
				h(ctx)
			}
		}
	})
}

// middleware - validate data when user saves listing to a folder
func ValidateAddUserListing(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		helper := ellimango.Helper{Env: env}

		// validate listing id, listing affiliate_id, folder name
		listingTpid := strings.ToUpper(getFormValue(ctx, "listing_id"))
		affiliateId := getFormValue(ctx, "listing_aff_id")
		folderName := getFormValue(ctx, "folder_name")
		missingFields := make([]string, 0)

		if listingTpid == "" {
			missingFields = append(missingFields, "listing_id")
		}

		if affiliateId == "" {
			missingFields = append(missingFields, "listing_aff_id")
		}

		if folderName == "" {
			missingFields = append(missingFields, "folder_name")
		}

		// some fields are empty
		if len(missingFields) > 0 {
			// outputResponse(ctx, 400, ellimango.Response{Reason: PLS_FILL_FIELDS + strings.Join(missingFields, ",")})
			outputResponse(ctx, 400, ellimango.Response{Reason: LST_NOT_SAVED})
			// required fields are not empty
		} else {
			dbAffiliateId := helper.ConvertEllimanAffiliateIdToDbAffiliateId(affiliateId)
			// affiliate id is invalid
			if dbAffiliateId == "" {
				outputResponse(ctx, 400, ellimango.Response{Reason: "Invalid listing_aff_id: " + affiliateId})
			} else {
				// get listing by listing third party id and db affiliate id
				oracle, redis, _ := getOracleRedisRabbitmqHelpers()
				listing := ellimango.Listing{Oracle: oracle,
					Redis: redis,
				}
				listingId, errorMessage := listing.GetListingIdByTpidAndAffId(listingTpid, dbAffiliateId)
				helper.Debug("ValidateAddUserListing listingId=" + listingId + ", errorMessage=" + errorMessage)

				// error on GetListingIdByTpidAndAffId
				if errorMessage != "" {
					outputResponse(ctx, 500, ellimango.Response{Reason: errorMessage})
					// no error, but listing not found
					// listing not found by third party id and affiliate id
				} else if listingId == "" {
					outputResponse(ctx, 404, ellimango.Response{Reason: LST_NOT_FOUND})
					// listing found
				} else {
					// get folder id by folder name and user id
					userId := ctx.UserValue("userId").(string)
					folder := ellimango.Folder{Oracle: oracle,
						Redis: redis,
					}
					folderId, errorMessage := folder.GetListingFolderIdByUserIdAndFolderName(userId, folderName)
					helper.Debug("ValidateAddUserListing folderId=" + folderId + ", errorMessage=" + errorMessage)
					// error on GetFolderIdByUserIdAndFolderName
					if errorMessage != "" {
						outputResponse(ctx, 500, ellimango.Response{Reason: errorMessage})
						// no error
					} else {
						// folder does not exist, we need to create it
						if folderId == "" {
							iFolderId, errorMessage := folder.CreateListingFolderWithUserIdAndFolderName(userId, folderName)
							// folder was not created
							if errorMessage != "" {
								outputResponse(ctx, 500, ellimango.Response{Reason: errorMessage})
								// folder was created
							} else if errorMessage == "" && iFolderId > 0 {
								folderId = strconv.FormatUint(iFolderId, 10)
							}
						}

						// we reach this point, when there are no errors
						if listingId != "" && folderId != "" {
							ctx.SetUserValue("listingId", listingId)
							ctx.SetUserValue("folderId", folderId)
							h(ctx)
						}
					}
				}
			}
		}
	})
}

// middleware - validate data when user removes listing from a folder
func ValidateDeleteUserListing(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		helper := ellimango.Helper{Env: env}

		// validate listing id, listing affiliate_id, folder name
		listingTpid := strings.ToUpper(getFormValue(ctx, "listing_id"))
		affiliateId := getFormValue(ctx, "listing_aff_id")
		folderName := getFormValue(ctx, "folder_name")
		missingFields := make([]string, 0)

		if listingTpid == "" {
			missingFields = append(missingFields, "listing_id")
		}

		if affiliateId == "" {
			missingFields = append(missingFields, "listing_aff_id")
		}

		if folderName == "" {
			missingFields = append(missingFields, "folder_name")
		}

		// some fields are empty
		if len(missingFields) > 0 {
			outputResponse(ctx, 400, ellimango.Response{Reason: PLS_FILL_FIELDS + strings.Join(missingFields, ",")})
			// required fields are not empty
		} else {
			dbAffiliateId := helper.ConvertEllimanAffiliateIdToDbAffiliateId(affiliateId)
			// affiliate id is invalid
			if dbAffiliateId == "" {
				outputResponse(ctx, 400, ellimango.Response{Reason: "Invalid listing_aff_id: " + affiliateId})
			} else {
				// get listing by listing third party id and db affiliate id
				oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
				listing := ellimango.Listing{Oracle: oracle,
					Redis: redis,
				}
				listingId, errorMessage := listing.GetListingIdByTpidAndAffId(listingTpid, dbAffiliateId)
				helper.Debug("ValidateDeleteUserListing listingId=" + listingId + ", errorMessage=" + errorMessage)

				// error on GetListingIdByTpidAndAffId
				if errorMessage != "" {
					outputResponse(ctx, 500, ellimango.Response{Reason: errorMessage})
					// no error, but listing not found
					// listing not found by third party id and affiliate id
				} else if listingId == "" {
					outputResponse(ctx, 404, ellimango.Response{Reason: LST_NOT_FOUND})
					// listing found
				} else {
					// get folder id by folder name and user id
					userId := ctx.UserValue("userId").(string)
					folder := ellimango.Folder{Oracle: oracle,
						Redis: redis,
					}
					folderId, errorMessage := folder.GetListingFolderIdByUserIdAndFolderName(userId, folderName)
					helper.Debug("ValidateDeleteUserListing folderId=" + folderId + ", errorMessage=" + errorMessage)
					// error on GetFolderIdByUserIdAndFolderName
					if errorMessage != "" {
						outputResponse(ctx, 500, ellimango.Response{Reason: errorMessage})
						// no error
					} else {
						// folder does not exist
						if folderId == "" {
							outputResponse(ctx, 404, ellimango.Response{Reason: FOLDER_NOT_FOUND})
							// folder found
						} else {
							user := ellimango.User{Oracle: oracle,
								Redis: redis,
							}
							userWithP, err := user.GetByUserId(userId)
							// no error from redis or oracle
							if err == nil {
								// user not found neither in redis nor in oracle
								if userWithP.DeUserId == "" {
									// outputResponse(ctx, 404, ellimango.Response{Reason: userId + " not found"})
									outputResponse(ctx, 404, ellimango.Response{Reason: USER_NOT_FOUND})
									// user found either in redis or oracle
								} else {
									// listing found, folder found, user found
									// lets make sure that user has this listing in the folder
									// we reach this point, when there are no errors
									userListing := ellimango.UserListing{
										Oracle:   oracle,
										Redis:    redis,
										Rabbitmq: rabbitmq,
									}
									found := userListing.Find(userId, folderId, listingId)
									if found == true {
										ctx.SetUserValue("listingId", listingId)
										ctx.SetUserValue("folderId", folderId)
										h(ctx)
									} else {
										outputResponse(ctx, 404, ellimango.Response{Reason: "User " + userId + " does not have listing " + listingTpid + " in " + folderName})
									}
								}
							} else {
								outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
							}
						}
					}
				}
			}
		}
	})
}

// helper - to output response as json
func outputResponse(ctx *fasthttp.RequestCtx, statusCode int, response interface{}) {
	ctx.Response.Header.Set("Server", "Elliman api server")
	js, err := json.Marshal(response)
	if err == nil {
		ctx.SetStatusCode(statusCode)
		ctx.SetContentType("application/json; charset=utf8")
		ctx.SetBody(js)
	} else {
		ctx.SetStatusCode(500)
		log.Println("outputResponse json marshall error=", err, ",response=", response)
		fmt.Fprint(ctx, PLS_TRY_LATER)
	}
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

// helper - to get form value by key
func getFormValue(ctx *fasthttp.RequestCtx, key string) string {
	var value string
	contentType := strings.ToLower(string(ctx.Request.Header.ContentType()))

	//Content-Type can actually contain a '; charset=...' part; ignore this if present
	//eg: application/x-www-form-urlencoded; charset=utf-8
	components := strings.SplitN(contentType, ";", 2)
	if len(components) >= 2 {
		contentType = components[0]
	}

	if contentType == "application/x-www-form-urlencoded" {
		// ex. curl -X POST -H "AuthToken: rafikhelloworld" -H "Content-Type: application/x-www-form-urlencoded" -d "password=ë,ï,ü,â,ê,î,ô,û,à,è,ù,é,ç,рафик&email=value2"
		v := ctx.PostArgs().Peek(key)
		if len(v) > 0 {
			value = string(v)
		}
	} else {
		mf, err := ctx.MultipartForm()
		if err == nil && mf.Value != nil {
			vv := mf.Value[key]
			if len(vv) > 0 {
				// ex. curl -X POST -H "AuthToken: rafikhelloworld" -H "Content-Type: multipart/form-data" -F "email=dasha@smarteragent.com" http://192.168.50.81:81/v1/user/login -F "password=ë,ï,ü,â,ê,î,ô,û,à,è,ù,é,ç"
				value = vv[0]
			}
		} else {
			// ex. curl -X POST -H "AuthToken: rafikhelloworld" -H "Content-Type: application/json" -d '{"password":"рафик,ë,ï,ü,â,ê,î,ô,û,à,è,ù,é,ç","email":"rafik@test.com"}' http://192.168.50.81:81/v1/user/login
			var reqBody interface{}
			err := json.Unmarshal(ctx.PostBody(), &reqBody)
			if err != nil {
				log.Println("reqBody unmarshall error", err)
			} else {
				reqBodyMap := reqBody.(map[string]interface{})
				if reqBodyMap[key] != nil {
					value = fmt.Sprintf("%v", reqBodyMap[key])
				}
				// for kkk, vvv := range reqBodyMap {
				// 	if kkk == key && vvv != nil {
				// 		str := fmt.Sprintf("%v", vvv)
				// 		value = str
				// 	}
				// }
			}
		}
	}
	return value
}

// handler - get all dual agents
// to test - curl -k -H 'Authtoken: rafikhelloworld' -H 'Accept-Language: es' -H 'Cookie: ID=1234' http://192.168.50.81:port/v1/dualagents
func GetDualAgents(ctx *fasthttp.RequestCtx) {
	oracle, redis, _ := getOracleRedisRabbitmqHelpers()
	da := ellimango.DualAgents{Oracle: oracle,
		Redis: redis,
	}

	dualAgents := da.Get()
	// loop thru data
	// do some manipulations
	// output as json
	if len(dualAgents) > 0 {
		agents := make([]ellimango.DualAgent, len(dualAgents))
		agentNum := 0
		for pId, sIds := range dualAgents {

			sIdsNum := len(sIds)
			secondaryAgentIds := make([]string, sIdsNum)
			for index, sId := range sIds {
				secondaryAgentIds[index] = sId

			}
			agents[agentNum] = ellimango.DualAgent{PrimaryAgentId: pId, SecondaryAgentId: secondaryAgentIds}
			agentNum = agentNum + 1
		}
		outputResponse(ctx, 200, agents)
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - get user by userId
func GetUser(ctx *fasthttp.RequestCtx) {
	userId := ctx.UserValue("userId").(string)

	oracle, redis, _ := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}

	userWithP, err := user.GetByUserId(userId)
	// no error from redis or oracle
	if err == nil {
		// user not found neither in redis nor in oracle
		if userWithP.DeUserId == "" {
			// outputResponse(ctx, 404, ellimango.Response{Reason: userId + " not found"})
			outputResponse(ctx, 404, ellimango.Response{Reason: USER_NOT_FOUND})
			// user found either in redis or oracle
		} else {
			userWithP.P = ""
			userWithP.ExpiresOnTs = 0
			outputResponse(ctx, 200, userWithP)
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - get user by email and password
func AuthenticateUser(ctx *fasthttp.RequestCtx) {
	email := ctx.UserValue("email").(string)
	password := ctx.UserValue("password").(string)

	oracle, redis, _ := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}

	userWithP, err := user.GetByEmailAndPassword(email, password)
	// no error from redis or oracle
	if err == nil {
		// user not found in redis and oracle
		if userWithP.DeUserId == "" {
			outputResponse(ctx, 404, ellimango.Response{Reason: INCORRECT_EMAIL_OR_PASS})
			// user found either in redis or oracle
		} else {
			if userWithP.EmailVerified == 1 && userWithP.ExpiresOnTs >= time.Now().Unix() {
				userWithP.P = ""
				userWithP.ExpiresOnTs = 0
				outputResponse(ctx, 200, userWithP)
			} else {
				if userWithP.EmailVerified != 1 {
					outputResponse(ctx, 403, ellimango.UserShort{Reason: EMAIL_NOT_VERIFIED, DeUserId: userWithP.DeUserId})
				} else if userWithP.ExpiresOnTs < time.Now().Unix() {
					outputResponse(ctx, 403, ellimango.UserShort{Reason: ACC_HAS_EXPIRED, DeUserId: userWithP.DeUserId})
				}
			}
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - create user
func CreateUser(ctx *fasthttp.RequestCtx) {
	// "firstname": "string", required
	// "lastname": "string", required
	// "email": "string", required
	// "password": "string", required, 4 char min
	// "sa_app_code": "string", required
	firstname := ctx.UserValue("firstname").(string)
	lastname := ctx.UserValue("lastname").(string)
	email := ctx.UserValue("email").(string)
	phone := ctx.UserValue("phone").(string)
	password := ctx.UserValue("password").(string)
	saAppCode := ctx.UserValue("saAppCode").(string)
	deAgentId := ctx.UserValue("deAgentId").(string)
	fbId := ctx.UserValue("fbId").(string)
	originIp := ctx.UserValue("originIp").(string)
	agentId := ctx.UserValue("agentId").(uint64)

	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis:    redis,
		Rabbitmq: rabbitmq,
	}

	userWithP, errorMessage := user.Create(firstname, lastname, email, phone, password, saAppCode, deAgentId, fbId, originIp, agentId)
	// user not created in oracle
	if errorMessage != "" {
		outputResponse(ctx, 400, ellimango.Response{Reason: errorMessage})
		// user is created in oracle
	} else if userWithP.DeUserId != "" {
		outputResponse(ctx, 200, ellimango.UserShort{DeUserId: userWithP.DeUserId})
	}
}

// handler - update user
func UpdateUser(ctx *fasthttp.RequestCtx) {
	// "firstname": "string", required
	// "lastname": "string", required
	// "email": "string", required
	// "password": "string", required, 8 char min
	// "sa_app_code": "string", required
	firstname := ctx.UserValue("firstname").(string)
	lastname := ctx.UserValue("lastname").(string)
	email := ctx.UserValue("email").(string)
	phone := ctx.UserValue("phone").(string)
	password := ctx.UserValue("password").(string)
	saAppCode := ctx.UserValue("saAppCode").(string)
	deAgentId := ctx.UserValue("deAgentId").(string)
	fbId := ctx.UserValue("fbId").(string)
	originIp := ctx.UserValue("originIp").(string)
	agentId := ctx.UserValue("agentId").(uint64)
	userId := ctx.UserValue("userId").(string)
	proceed := false

	// need to meet 2 requirements
	// user with userId must exist
	// another user with email must not exist
	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis:    redis,
		Rabbitmq: rabbitmq,
	}
	// find user by userId
	userWithP, err := user.GetByUserId(userId)
	// no error from redis or oracle
	if err == nil {
		// user not found neither in redis nor in oracle
		if userWithP.DeUserId == "" {
			// outputResponse(ctx, 404, ellimango.Response{Reason: userId + " not found"})
			outputResponse(ctx, 404, ellimango.Response{Reason: USER_NOT_FOUND})
			// user found either in redis or oracle
		} else {
			userIdByEmail, err := user.GetUserIdByEmail(email)
			// error
			if err != nil {
				outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
				// no error
			} else {
				// user with such email found
				if userIdByEmail != "" {
					// same user
					if userId == userIdByEmail {
						proceed = true
						// another user
					} else {
						outputResponse(ctx, 400, ellimango.Response{Reason: EMAIL_EXISTS})
					}
					// user with such email not found
				} else {
					proceed = true
				}
				if proceed == true {
					updatedUserWithP, errorMessage := user.Update(firstname, lastname, email, phone, password, saAppCode, deAgentId, fbId, originIp, agentId, userWithP)
					// user not updated
					if errorMessage != "" {
						outputResponse(ctx, 400, ellimango.Response{Reason: errorMessage})
						// user updated
					} else if updatedUserWithP.DeUserId != "" {
						outputResponse(ctx, 200, ellimango.UserShort{DeUserId: updatedUserWithP.DeUserId})
					}
				}
			}
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - update user password expiration date
func ResetUserExpiration(ctx *fasthttp.RequestCtx) {
	userId := ctx.UserValue("userId").(string)

	oracle, redis, _ := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}
	// find user by userId
	userWithP, err := user.GetByUserId(userId)
	// no error from redis or oracle
	if err == nil {
		// user not found neither in redis nor in oracle
		if userWithP.DeUserId == "" {
			outputResponse(ctx, 404, ellimango.Response{Reason: "We were not able to find a user record. Please contact support at info@elliman.com"})
			// user found either in redis or oracle, update user expiration
		} else {
			updatedUserWithP, errorMessage := user.UpdateUserExpiration(userId)
			// user not updated
			if errorMessage != "" {
				outputResponse(ctx, 400, ellimango.Response{Reason: errorMessage})
				// user updated
			} else if updatedUserWithP.DeUserId != "" {
				updatedUserWithP.P = ""
				updatedUserWithP.ExpiresOnTs = 0
				outputResponse(ctx, 200, updatedUserWithP)
			}
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - get user listings
func GetUserListings(ctx *fasthttp.RequestCtx) {
	userId := ctx.UserValue("userId").(string)

	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}
	// find user by userId
	userWithP, err := user.GetByUserId(userId)
	// no error from redis or oracle
	if err == nil {
		// user not found neither in redis nor in oracle
		if userWithP.DeUserId == "" {
			// outputResponse(ctx, 404, ellimango.Response{Reason: userId + " not found"})
			outputResponse(ctx, 404, ellimango.Response{Reason: USER_NOT_FOUND})
			// user found either in redis or oracle, get user all listings
		} else {
			userListing := ellimango.UserListing{
				Oracle:   oracle,
				Redis:    redis,
				Rabbitmq: rabbitmq,
			}
			userListings, errorMessage := userListing.GetAll(userId)
			if errorMessage != "" {
				outputResponse(ctx, 400, ellimango.Response{Reason: errorMessage})
				// no error on getting user listings
			} else {
				outputResponse(ctx, 200, userListings)
			}
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - add user listing
func AddUserListing(ctx *fasthttp.RequestCtx) {
	userId := ctx.UserValue("userId").(string)
	listingId := ctx.UserValue("listingId").(string)
	folderId := ctx.UserValue("folderId").(string)

	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}
	// find user by userId
	userWithP, err := user.GetByUserId(userId)

	// no error from redis or oracle
	if err == nil {
		// user not found neither in redis nor in oracle
		if userWithP.DeUserId == "" {
			// outputResponse(ctx, 404, ellimango.Response{Reason: userId + " not found"})
			outputResponse(ctx, 404, ellimango.Response{Reason: USER_NOT_FOUND})
			// user found either in redis or oracle, add listing
		} else {
			userListing := ellimango.UserListing{
				Oracle:   oracle,
				Redis:    redis,
				Rabbitmq: rabbitmq,
			}
			userListings, errorMessage := userListing.Add(userId, folderId, listingId)
			// listing not saved
			if errorMessage != "" {
				outputResponse(ctx, 500, ellimango.Response{Reason: errorMessage})
				// listing saved
			} else {
				outputResponse(ctx, 200, userListings)
			}
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - user has to receive email with confirmation link
func VerifyUser(ctx *fasthttp.RequestCtx) {
	userId := ctx.UserValue("userId").(string)

	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis:    redis,
		Rabbitmq: rabbitmq,
	}

	userWithP, err := user.GetByUserId(userId)
	// no error from redis or oracle
	if err == nil {
		// user not found neither in redis nor in oracle
		if userWithP.DeUserId == "" {
			// outputResponse(ctx, 404, ellimango.Response{Reason: userId + " not found"})
			outputResponse(ctx, 404, ellimango.Response{Reason: USER_NOT_FOUND})
			// user found either in redis or oracle
		} else {
			// we have to check that user has email_verified as 0
			if userWithP.EmailVerified == 1 {
				outputResponse(ctx, 500, ellimango.Response{Reason: "User " + userId + " has already verified email"})
			} else {
				// send data to rabbitmq
				user.RabbitmqMembershipVerificationEmail(userId)
				outputResponse(ctx, 200, ellimango.Response{Reason: "Email sent to " + userWithP.Email})
			}
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

// handler - delete user listing
func DeleteUserListing(ctx *fasthttp.RequestCtx) {
	userId := ctx.UserValue("userId").(string)
	listingId := ctx.UserValue("listingId").(string)
	folderId := ctx.UserValue("folderId").(string)

	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	userListing := ellimango.UserListing{
		Oracle:   oracle,
		Redis:    redis,
		Rabbitmq: rabbitmq,
	}
	userListings, errorMessage := userListing.Delete(userId, folderId, listingId)
	// listing not deleted
	if errorMessage != "" {
		outputResponse(ctx, 500, ellimango.Response{Reason: errorMessage})
		// listing deleted
	} else {
		outputResponse(ctx, 200, userListings)
	}
}

// handler - get actual health stats
func GetHealth(ctx *fasthttp.RequestCtx) {
	oracle, redis, rabbitmq := getOracleRedisRabbitmqHelpers()
	helper := ellimango.Helper{Env: env}

	sysdateStr, oraErr := oracle.GetSysdate()
	pingStr, redisErr := redis.Ping()
	rabbitmqStatus, rabbitmqStatusErr := helper.GetRabbitmqStatus(rabbitmq)

	if oraErr == nil && redisErr == nil && rabbitmqStatusErr == nil {
		helper.Debug("Oracle sysdate=" + sysdateStr + ", redis ping=" + pingStr)
		helper.Debug("Rabbitmq status=", rabbitmqStatus)
		outputResponse(ctx, 200, ellimango.Response{Reason: "Healthy", Version: ellimango.VERSION})
	} else {
		reasons := make([]string, 0)
		if oraErr != nil {
			reasons = append(reasons, fmt.Sprintf("Oracle error: %v", oraErr))
		}
		if redisErr != nil {
			reasons = append(reasons, fmt.Sprintf("Redis error: %v", redisErr))
		}
		if rabbitmqStatusErr != nil {
			reasons = append(reasons, fmt.Sprintf("Rabbitmq error: %v", rabbitmqStatusErr))
		}

		outputResponse(ctx, 500, ellimango.Response{Reason: strings.Join(reasons, "|")})
	}
}

// handler - get user by email
func GetUserByEmail(ctx *fasthttp.RequestCtx) {
	email := strings.ToLower(ctx.UserValue("email").(string))

	oracle, redis, _ := getOracleRedisRabbitmqHelpers()
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}

	userWithP, err := user.GetByEmail(email)
	// no error from redis or oracle
	if err == nil {
		// user not found neither in redis nor in oracle
		if userWithP.DeUserId == "" {
			// outputResponse(ctx, 404, ellimango.Response{Reason: userId + " not found"})
			outputResponse(ctx, 404, ellimango.Response{Reason: USER_NOT_FOUND})
			// user found either in redis or oracle
		} else {
			userWithP.P = ""
			userWithP.ExpiresOnTs = 0
			outputResponse(ctx, 200, userWithP)
		}
	} else {
		outputResponse(ctx, 500, ellimango.Response{Reason: PLS_TRY_LATER})
	}
}

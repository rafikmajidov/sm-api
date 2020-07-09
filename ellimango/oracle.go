package ellimango

import (
	"encoding/json"
	"fmt"
	"github.com/satori/go.uuid"
	"gopkg.in/rana/ora.v3"
	"log"
	"strconv"
	"strings"
	"time"
)

type Oracle struct {
	Env     string
	OraUser string
	OraPass string
	OraSid  string
}

// connect to oracle
func (oracle *Oracle) Connect() (*ora.Env, *ora.Srv, *ora.Ses, error) {
	oraString := oracle.OraUser + "/" + oracle.OraPass + "@" + oracle.OraSid
	oraEnvCfg := ora.NewEnvCfg()
	oraEnv, oraSrv, oraSes, err := ora.NewEnvSrvSes(oraString, oraEnvCfg)
	if err != nil {
		log.Println("Oracle connect error", err)
		//go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Oracle connect error %v", err), "Oracle connect error")
	}
	return oraEnv, oraSrv, oraSes, err
}

// close connection
func (oracle *Oracle) Close(oraEnv *ora.Env, oraSrv *ora.Srv, oraSes *ora.Ses) {
	//helper := Helper{Env: oracle.Env}

	err := oraEnv.Close()
	if err != nil {
		log.Println("Oracle env close error", err)
	} else {
		//helper.Debug("Oracle env close error", err)
	}
	err = oraSrv.Close()
	if err != nil {
		log.Println("Oracle server close error", err)
	} else {
		//helper.Debug("Oracle server close error", err)
	}
	err = oraSes.Close()
	if err != nil {
		log.Println("Oracle session close error", err)
	} else {
		//helper.Debug("Oracle session close error", err)
	}
}

// get all rows from fo_combined_agt
func (oracle *Oracle) GetDualAgents() (map[string]map[int]string, []string) {
	pIdSids := make([]string, 0)
	dualAgents := make(map[string]map[int]string)

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	if err == nil {
		qry := "SELECT * FROM fo_combined_agt "
		rset, err := oraSes.PrepAndQry(qry)
		if err != nil {
			log.Println("Oracle error with query=", qry, ", error=", err)
		} else {
			var sIdLen int
			for rset.Next() {
				pId := rset.Row[0].(string)
				sId := rset.Row[1].(string)
				pIdSid := pId + ":" + sId
				pIdSids = append(pIdSids, pIdSid)

				// create array only once
				if dualAgents[pId] == nil {
					dualAgents[pId] = make(map[int]string)
				}

				sIdLen = len(dualAgents[pId])
				dualAgents[pId][sIdLen] = sId
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return dualAgents, pIdSids
}

// get data from fo_user_info by id
// return userWithP
func (oracle *Oracle) GetUserByUserId(userId string) (UserWithP, error) {
	var userWithP UserWithP

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	if err == nil {
		// both queries in get user and authenticate user must be the same
		qry := "SELECT fui.first_name, fui.last_name, fui.email_address, fui.day_phone_number, fue.sa_app_code, fai.third_party_id, fui.facebook_id, fui.ip_address, ut_date_to_unix(fui.date_password_expires) date_password_expires, fui.email_is_verified, fui.id de_user_id, fui.password  FROM fo_user_info fui left join fo_user_agent fua on fua.user_id=fui.id left join fo_user_extra fue on fue.user_id=fui.id left join fo_agt_info fai on fai.id=fua.agent_id WHERE fui.id = :userId"
		rset, err2 := oraSes.PrepAndQry(qry, userId)
		if err2 != nil {
			log.Println("Oracle error with query=", qry, ", error=", err2, ", userId=", userId)
			userWithP = UserWithP{}
			err = err2
		} else {
			userWithP = oracle.GetUserFromRset(rset)
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}
	return userWithP, err
}

// get data from fo_user_info by email_address and password
// return userWithP
func (oracle *Oracle) GetUserByEmailAndPassword(email string, password string) (UserWithP, error) {
	var userWithP UserWithP

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	if err == nil {
		qry := "SELECT fui.first_name, fui.last_name, fui.email_address, fui.day_phone_number, fue.sa_app_code, fai.third_party_id, fui.facebook_id, fui.ip_address, ut_date_to_unix(fui.date_password_expires) date_password_expires, fui.email_is_verified, fui.id de_user_id, fui.password  FROM fo_user_info fui left join fo_user_agent fua on fua.user_id=fui.id left join fo_user_extra fue on fue.user_id=fui.id left join fo_agt_info fai on fai.id=fua.agent_id WHERE fui.email_address = :email and fui.password = :password"
		rset, err2 := oraSes.PrepAndQry(qry, email, password)
		if err2 != nil {
			log.Println("Oracle error with query=", qry, ", error=", err2)
			userWithP = UserWithP{}
			err = err2
		} else {
			userWithP = oracle.GetUserFromRset(rset)
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}
	return userWithP, err
}

// populate struct UserWithP from result set
func (oracle *Oracle) GetUserFromRset(rset *ora.Rset) UserWithP {
	var userWithP UserWithP

	for rset.Next() {
		firstname := rset.Row[0].(string)
		lastname := rset.Row[1].(string)
		email := rset.Row[2].(string)
		phone := rset.Row[3].(string)
		saAppCode := rset.Row[4].(string)
		deAgentId := rset.Row[5].(string)
		fbId := rset.Row[6].(string)
		originIp := rset.Row[7].(string)
		expiresOn := ""
		var expiresOnTs int64
		if rset.Row[8] != nil {
			//expiresOn = rset.Row[8].(time.Time).String()
			i, err := strconv.ParseInt(rset.Row[8].(ora.OCINum).String(), 10, 64)
			if err == nil {
				expiresOn = fmt.Sprintf("%v", time.Unix(i, 0))
				expiresOnTs = i
			}
		}
		emailVerified := rset.Row[9].(int64)
		deUserId := rset.Row[10].(ora.OCINum).String()
		p := rset.Row[11].(string)

		userWithP = UserWithP{DeUserId: deUserId,
			Firstname:     strings.ToLower(firstname),
			Lastname:      strings.ToLower(lastname),
			Email:         strings.ToLower(email),
			Phone:         phone,
			SaAppCode:     saAppCode,
			DeAgentId:     strings.ToUpper(deAgentId),
			FbId:          fbId,
			OriginIp:      originIp,
			ExpiresOn:     expiresOn,
			EmailVerified: emailVerified,
			P:             p,
			ExpiresOnTs:   expiresOnTs}
	}
	return userWithP
}

// get agent id by third party id
func (oracle *Oracle) GetAgentIdByThirdPartyId(deAgentId string) (uint64, error) {
	var agentId uint64

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	if err == nil {
		qry := "SELECT id FROM fo_agt_info WHERE upper(third_party_id) = :deAgentId and is_active = 1"
		rset, err2 := oraSes.PrepAndQry(qry, strings.ToUpper(deAgentId))
		if err2 != nil {
			log.Println("Oracle error with query=", qry, ", error=", err2)
			err = err2
		} else {
			for rset.Next() {
				agentId, err = strconv.ParseUint(rset.Row[0].(ora.OCINum).String(), 10, 64)
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return agentId, err
}

// get user id by email
func (oracle *Oracle) GetUserIdByEmail(email string) (string, error) {
	var userId string

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	if err == nil {
		qry := "SELECT id FROM fo_user_info WHERE email_address = :email"
		rset, err2 := oraSes.PrepAndQry(qry, email)
		if err2 != nil {
			log.Println("Oracle error with query=", qry, ", error=", err2, ", email=", email)
			err = err2
		} else {
			for rset.Next() {
				userId = rset.Row[0].(ora.OCINum).String()
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return userId, err
}

// insert into fo_user_info
// insert into fo_user_extra
// if agent is present insert into fo_user_agent
func (oracle *Oracle) CreateUser(firstname string, lastname string, email string, phone string, password string, saAppCode string, deAgentId string, fbId string, originIp string, agentId uint64) (UserWithP, string) {
	helper := Helper{Env: oracle.Env}
	var userWithP UserWithP
	var errorMessage = USER_NOT_CREATED
	var rollback = false

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		tx, err := oraSes.StartTx()
		// no transaction error
		if err == nil {
			var userId uint64
			uq := uuid.NewV4()
			identifier := uq.String()
			// insert data into fo_user_info
			sql := "INSERT INTO fo_user_info(id, account_status_id, day_phone_number, first_name, last_name, email_address, password, email_is_verified, date_created, ip_address, facebook_id, new_tomarket_email, price_reduction_email, identifier, date_password_expires) VALUES(fo_user_info_seq.nextval, 2, :phone, :firstname, :lastname, :email, :password, 0, sysdate, :originIp, :fbId, 0, 0, :identifier, sysdate + 120) RETURNING ID /*lastInsertId*/ INTO :C1"
			stmtIns, err := oraSes.Prep(sql)
			defer stmtIns.Close()
			rowsAffected, err := stmtIns.Exe(phone, firstname, lastname, email, password, originIp, fbId, identifier, &userId)
			// user not inserted
			if err != nil {
				log.Println("Oracle error with query=", sql, ", error=", err)
				tmpStr := strings.ToUpper(fmt.Sprintf("%v", err))
				if strings.Contains(tmpStr, "FO_USER_INFO_EMAIL_ADDRESS_UK") {
					errorMessage = EMAIL_EXISTS
				}
				rollback = true
				// user inserted
			} else if userId > 0 && rowsAffected == 1 {
				helper.Debug("Insert into fo_user_info", userId)
				// insert data into fo_user_extra
				sql := "INSERT INTO fo_user_extra(user_id, sa_app_code) VALUES(:userId, :saAppCode)"
				stmtIns, err := oraSes.Prep(sql)
				defer stmtIns.Close()
				rowsAffected, err = stmtIns.Exe(userId, saAppCode)
				if err != nil {
					log.Println("Oracle error with query=", sql, ", error=", err)
					rollback = true
				}

				// if agentId not empty insert into fo_user_agent
				if agentId > 0 {
					sql := "INSERT INTO fo_user_agent(user_id, agent_id) VALUES(:userId, :agentId)"
					stmtIns, err := oraSes.Prep(sql)
					defer stmtIns.Close()
					rowsAffected, err = stmtIns.Exe(userId, agentId)
					if err != nil {
						log.Println("Oracle error with query=", sql, ", error=", err)
						rollback = true
					}
				}
			}

			if rollback == true {
				tx.Rollback()
				helper.Debug("Oracle CreateUser Rollback")
			} else {
				tx.Commit()
				// userWithP = UserWithP{DeUserId: strconv.FormatUint(userId, 10),
				// 	Firstname:     firstname,
				// 	Lastname:      lastname,
				// 	Email:         email,
				// 	Phone:         phone,
				// 	SaAppCode:     saAppCode,
				// 	DeAgentId:     deAgentId,
				// 	FbId:          fbId,
				// 	OriginIp:      originIp,
				// 	ExpiresOn:     "",
				// 	EmailVerified: 0,
				// 	P:             password}
				// errorMessage = ""
				// helper.Debug("Oracle CreateUser Commit")
				// get created user by id
				userWithP, err = oracle.GetUserByUserId(strconv.FormatUint(userId, 10))
				errorMessage = ""
				helper.Debug("Oracle CreateUser Commit")
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}
	helper.Debug("oracle createuser err=", err)
	helper.Debug("oracle createuser userWithP=", userWithP)
	helper.Debug("oracle createuser errorMessage=" + errorMessage)
	return userWithP, errorMessage
}

// update fo_user_info
// update fo_user_extra
// update/delete/insert fo_user_agent
func (oracle *Oracle) UpdateUser(firstname string, lastname string, email string, phone string, password string, saAppCode string, deAgentId string, fbId string, originIp string, agentId uint64, existingUserWithP UserWithP) (UserWithP, string) {
	helper := Helper{Env: oracle.Env}
	var userWithP UserWithP
	var errorMessage = USER_NOT_UPDATED
	var rollback = false

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		tx, err := oraSes.StartTx()
		// no transaction error
		if err == nil {
			// update data in fo_user_info, if changed
			if existingUserWithP.Firstname != firstname || existingUserWithP.Lastname != lastname || existingUserWithP.Email != email || existingUserWithP.Phone != phone || existingUserWithP.FbId != fbId || existingUserWithP.OriginIp != originIp || existingUserWithP.P != password {
				sql := "UPDATE fo_user_info SET first_name = :firstname,last_name = :lastname,email_address = :email,day_phone_number = :phone,facebook_id = :fbId,ip_address = :originIp,password = :password  WHERE id = :userId"
				stmtIns, err := oraSes.Prep(sql)
				defer stmtIns.Close()
				rowsAffected, err := stmtIns.Exe(firstname, lastname, email, phone, fbId, originIp, password, existingUserWithP.DeUserId)
				// user not updated
				if err != nil {
					log.Println("Oracle error with query=", sql, ", error=", err)
					rollback = true
				} else {
					helper.Debug("Update fo_user_info ok", rowsAffected, sql)
				}
			}
			// update data in fo_user_extra, if changed
			if existingUserWithP.SaAppCode != saAppCode {
				sql := ""
				// add new saAppCode
				if existingUserWithP.SaAppCode == "" && saAppCode != "" {
					sql = "INSERT INTO fo_user_extra(sa_app_code, user_id) VALUES(:saAppCode, :userId)"

					// update saAppCode
				} else if existingUserWithP.SaAppCode != "" && saAppCode != "" {
					sql = "UPDATE fo_user_extra set sa_app_code = :saAppCode WHERE user_id = :userId"
				}
				stmtIns, err := oraSes.Prep(sql)
				defer stmtIns.Close()
				rowsAffected, err := stmtIns.Exe(saAppCode, existingUserWithP.DeUserId)

				if err != nil {
					log.Println("Oracle error with query=", sql, ", error=", err)
					rollback = true
				} else {
					helper.Debug("Insert/update fo_user_extra ok", rowsAffected, sql)
				}
			}
			// update/delete/add data in fo_user_agent, if changed
			if existingUserWithP.DeAgentId != deAgentId {
				sql := ""
				var stmtIns *ora.Stmt
				var rowsAffected uint64
				var err error
				// remove agent from user
				if existingUserWithP.DeAgentId != "" && deAgentId == "" {
					sql = "DELETE FROM fo_user_agent WHERE user_id = :userId"
					stmtIns, err = oraSes.Prep(sql)
					rowsAffected, err = stmtIns.Exe(existingUserWithP.DeUserId)

					// assign new agent to user
				} else if existingUserWithP.DeAgentId == "" && deAgentId != "" && agentId > 0 {
					sql = "INSERT INTO fo_user_agent(user_id, agent_id) VALUES(:userId, :agentId)"
					stmtIns, err = oraSes.Prep(sql)
					rowsAffected, err = stmtIns.Exe(existingUserWithP.DeUserId, agentId)

					// update agent of user
				} else if existingUserWithP.DeAgentId != "" && deAgentId != "" && agentId > 0 {
					sql = "UPDATE fo_user_agent set agent_id = :agentId WHERE user_id = :userId"
					stmtIns, err = oraSes.Prep(sql)
					rowsAffected, err = stmtIns.Exe(agentId, existingUserWithP.DeUserId)
				}

				defer stmtIns.Close()

				if err != nil {
					log.Println("Oracle error with query=", sql, ", error=", err)
					rollback = true
				} else {
					helper.Debug("Insert/update,delete fo_user_agent ok", rowsAffected, sql)
				}
			}

			if rollback == true {
				tx.Rollback()
				helper.Debug("Oracle UpdateUser Rollback")
			} else {
				tx.Commit()
				// get updated user by id
				userWithP, err = oracle.GetUserByUserId(existingUserWithP.DeUserId)
				errorMessage = ""
				helper.Debug("Oracle UpdateUser Commit")
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return userWithP, errorMessage
}

// update fo_user_info set date_password_expires = now + 120 days
func (oracle *Oracle) UpdateUserExpiration(userId string) (UserWithP, string) {
	helper := Helper{Env: oracle.Env}
	var userWithP UserWithP
	var errorMessage = USER_EXP_NOT_UPDATED

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		// update data in fo_user_info,
		sql := "UPDATE fo_user_info set date_password_expires = sysdate + 120 where id = :userId"
		stmtIns, err := oraSes.Prep(sql)
		defer stmtIns.Close()
		rowsAffected, err := stmtIns.Exe(userId)
		// not updated
		if err != nil {
			log.Println("Oracle error with query=", sql, ", error=", err)
		} else {
			helper.Debug("Update date_password_expires ok", rowsAffected, sql)
			// get updated user by id
			userWithP, err = oracle.GetUserByUserId(userId)
			errorMessage = ""
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return userWithP, errorMessage
}

// get listings saved by user
func (oracle *Oracle) GetUserListings(userId string) (SavedListings, string) {
	helper := Helper{Env: oracle.Env}
	var dbUserListings SavedListings
	var errorMessage = ""

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		// get all user folders
		folders := make([]SavedFolder, 0)
		qry := "SELECT pslf.id folder_id, pslf.name folder_name, pslf.source_id FROM prof_sav_listing_folder pslf WHERE user_id = :userId ORDER BY pslf.name ASC"
		rset, err := oraSes.PrepAndQry(qry, userId)

		if err != nil {
			log.Println("Oracle error with query=", qry, ", error=", err)
			errorMessage = CANT_GET_USER_LISTINGS
		} else {
			for rset.Next() {
				var folder SavedFolder
				folder.Id, err = strconv.ParseUint(rset.Row[0].(ora.OCINum).String(), 10, 64)
				folder.Name = rset.Row[1].(string)
				folder.SourceId, err = strconv.ParseUint(rset.Row[2].(ora.OCINum).String(), 10, 64)
				folder.Listings = make([]SavedListing, 0)
				folders = append(folders, folder)
			}
		}
		// user has saved listings folders
		// get listings of folders
		if len(folders) > 0 {
			foldersLen := len(folders)
			for j := 0; j < foldersLen; j++ {
				qry := "SELECT fli.third_party_id, fli.affiliate_id, fli.sub_affiliate_id FROM prof_saved_listing psl, fo_lst_info fli WHERE psl.folder_id = :folderId and fli.id = psl.listing_id and fli.access_type_id = 1"
				rset, err := oraSes.PrepAndQry(qry, folders[j].Id)

				if err != nil {
					log.Println("Oracle error with query=", qry, ", error=", err)
					errorMessage = CANT_GET_USER_LISTINGS
				} else {
					folders[j].Id = 0
					for rset.Next() {
						listingTpid := rset.Row[0].(string)
						affiliateId := helper.ConvertDbAffiliateIdToEllimanAffiliateId(rset.Row[1].(ora.OCINum).String())
						subAffiliateId := ""
						if rset.Row[2] != nil {
							subAffiliateId = rset.Row[2].(ora.OCINum).String()
						}
						// create listing
						listing := SavedListing{ListingTpid: listingTpid, AffiliateId: affiliateId, SubAffiliateId: subAffiliateId}
						folders[j].Listings = append(folders[j].Listings, listing)
					}
				}
			}
		}

		dbUserListings = SavedListings{DeUserId: userId, Folders: folders}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
		// connection error
	} else {
		errorMessage = CANT_GET_USER_LISTINGS
	}

	helper.Debug("Oracle dbUserListings=", dbUserListings)
	helper.Debug("Oracle errorMessage=", errorMessage)

	return dbUserListings, errorMessage
}

// get listing id by third party id and affiliate id
func (oracle *Oracle) GetListingIdByTpidAndAffId(listingTpid string, dbAffiliateId string) (string, string) {
	var listingId string
	var errorMessage string

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		qry := "SELECT id FROM fo_lst_info WHERE third_party_id = :listingTpid and affiliate_id = :dbAffiliateId and access_type_id = 1"
		rset, err := oraSes.PrepAndQry(qry, listingTpid, dbAffiliateId)
		if err != nil {
			log.Println("Oracle error with query=", qry, " ,error=", err, ", listingTpid=", listingTpid, ", dbAffiliateId=", dbAffiliateId)
			errorMessage = CANT_GET_LISTING
		} else {
			for rset.Next() {
				listingId = rset.Row[0].(ora.OCINum).String()
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
		// connection error
	} else {
		errorMessage = CANT_GET_LISTING
	}

	return listingId, errorMessage
}

// get listing folder id by user id and folder name
func (oracle *Oracle) GetListingFolderIdByUserIdAndFolderName(userId string, folderName string) (string, string) {
	var folderId string
	var errorMessage string

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		qry := "SELECT id FROM prof_sav_listing_folder WHERE user_id = :userId and name = :folderName"
		rset, err := oraSes.PrepAndQry(qry, userId, folderName)
		if err != nil {
			log.Println("Oracle error with query=", qry, ", error=", err, ", userId=", userId, ", folderName=", folderName)
			errorMessage = CANT_GET_FOLDER
		} else {
			for rset.Next() {
				folderId = rset.Row[0].(ora.OCINum).String()
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
		// connection error
	} else {
		errorMessage = CANT_GET_FOLDER
	}

	return folderId, errorMessage
}

// create listing folder
func (oracle *Oracle) CreateListingFolderWithUserIdAndFolderName(userId string, folderName string) (uint64, string) {
	helper := Helper{Env: oracle.Env}
	var folderId uint64
	var sourceId = "2"
	var errorMessage = FOLDER_NOT_CREATED

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		// insert data into prof_sav_listing_folder
		sql := "INSERT INTO prof_sav_listing_folder(id, user_id, name, date_created, source_id) VALUES(prof_sav_listing_folder_seq.nextval, :userId, :folderName, sysdate, :sourceId) RETURNING ID /*lastInsertId*/ INTO :C1"
		stmtIns, err := oraSes.Prep(sql)
		defer stmtIns.Close()
		rowsAffected, err := stmtIns.Exe(userId, folderName, sourceId, &folderId)
		if err != nil {
			log.Println("Oracle error with query=", sql, ", error=", err, ", userId=", userId, ", folderName=", folderName, "sourceId=", sourceId)
			// to do, we can check for specific oracle error
			// and set errorMessage to something like, such folder already exists
		} else {
			helper.Debug("Insert into prof_sav_listing_folder ok ", rowsAffected)
			errorMessage = ""
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return folderId, errorMessage
}

// create user saved listing
func (oracle *Oracle) CreateUserSavedListing(userId string, folderId string, listingId string) string {
	helper := Helper{Env: oracle.Env}
	var errorMessage = LST_NOT_SAVED

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		// insert data into prof_saved_listing
		sql := "INSERT INTO prof_saved_listing(user_id, folder_id, listing_id, date_created) VALUES(:userId, :folderId, :listingId, sysdate)"
		stmtIns, err := oraSes.Prep(sql)
		defer stmtIns.Close()
		rowsAffected, err := stmtIns.Exe(userId, folderId, listingId)
		if err != nil {
			log.Println("Oracle error with query=", sql, ", error=", err, ", userId=", userId, " folderId=", folderId, ", listingId=", listingId)
			tmpStr := strings.ToUpper(fmt.Sprintf("%v", err))
			if strings.Contains(tmpStr, "PROF_SAVED_LISTING_PK") {
				errorMessage = LST_ALREADY_SAVED
			}
		} else {
			helper.Debug("Insert into prof_saved_listing ok ", rowsAffected)
			errorMessage = ""
		}

		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return errorMessage
}

func (oracle *Oracle) GetAgentByAgentId(agentId uint64) (FoAgtInfo, error) {
	var foAgtInfo FoAgtInfo

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		qry := "SELECT id, em_address, third_party_id, first_name, last_name  FROM fo_agt_info WHERE id = :agentId"
		rset, err2 := oraSes.PrepAndQry(qry, agentId)
		if err2 != nil {
			log.Println("Oracle error with query=", qry, " ,error=", err2, ", agentId=", agentId)
			err = err2
		} else {
			for rset.Next() {
				foAgtInfo = FoAgtInfo{Id: rset.Row[0].(ora.OCINum).String(),
					Email:     rset.Row[1].(string),
					Tpid:      rset.Row[2].(string),
					Firstname: rset.Row[3].(string),
					Lastname:  rset.Row[4].(string),
				}

			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return foAgtInfo, err
}

// get listing folder name by folder id
func (oracle *Oracle) GetListingFolderNameByFolderId(folderId string) (string, string) {
	var folderName string
	var errorMessage string

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		qry := "SELECT name FROM prof_sav_listing_folder WHERE id = :folderId"
		rset, err := oraSes.PrepAndQry(qry, folderId)
		if err != nil {
			log.Println("Oracle error with query=", qry, ", error=", err, ", folderId=", folderId)
			errorMessage = CANT_GET_FOLDER
		} else {
			for rset.Next() {
				folderName = rset.Row[0].(string)
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
		// connection error
	} else {
		errorMessage = CANT_GET_FOLDER
	}

	return folderName, errorMessage
}

// find user saved listing
func (oracle *Oracle) FindUserSavedListing(userId string, folderId string, listingId string) bool {
	var found bool

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		qry := "SELECT user_id, folder_id, listing_id, date_created FROM prof_saved_listing WHERE user_id = :userId AND folder_id = :folderId AND listing_id = :listingId"
		rset, err := oraSes.PrepAndQry(qry, userId, folderId, listingId)
		if err != nil {
			log.Println("Oracle error with query=", qry, ", error=", err, ", userId=", userId, ", folderId=", folderId, ", listingId=", listingId)
		} else {
			for rset.Next() {
				found = true
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return found
}

// delete user saved listing
func (oracle *Oracle) DeleteUserSavedListing(userId string, folderId string, listingId string) string {
	helper := Helper{Env: oracle.Env}
	var errorMessage = LST_NOT_DELETED

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		// delete data into prof_saved_listing
		sql := "DELETE from prof_saved_listing WHERE user_id = :userId AND folder_id = :folderId AND listing_id = :listingId"
		stmtIns, err := oraSes.Prep(sql)
		defer stmtIns.Close()
		rowsAffected, err := stmtIns.Exe(userId, folderId, listingId)
		// err = errors.New("ora delete test error")
		if err != nil {
			log.Println("Oracle error with query=", sql, ", error=", err, ", userId=", userId, " folderId=", folderId, ", listingId=", listingId)
			// tmpStr := strings.ToUpper(fmt.Sprintf("%v", err))
			// if strings.Contains(tmpStr, "PROF_SAVED_LISTING_PK") {
			// 	errorMessage = LST_ALREADY_SAVED
			// }
		} else {
			helper.Debug("Delete from prof_saved_listing ok ", rowsAffected)
			errorMessage = ""
		}

		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return errorMessage
}

// get agent assigned to user
func (oracle *Oracle) GetAgentByUserId(userId string) (FoAgtInfo, error) {
	var foAgtInfo FoAgtInfo
	var foOfcInfo FoOfcInfo

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	// no connection error
	if err == nil {
		qry := "SELECT fai.id, fai.em_address, fai.third_party_id, fai.first_name, fai.last_name, fai.is_active, fai.phone_number, fai.mobile_number, fai.fax_number, fai.photo_url, foi.id, foi.third_party_id as office_tpid, foi.name, foi.address, ls.abbreviation, lc.ascii_name, luzc.value, frm.lcl_region FROM fo_agt_info fai, fo_user_agent fua, fo_ofc_info foi, loc_city lc, loc_state ls, loc_us_zip_code luzc, fo_ofc_region_map frm WHERE fua.user_id = :userId AND fua.agent_id = fai.id and foi.id = fai.office_id AND lc.id = foi.city_id AND ls.id = foi.state_id and luzc.id = foi.us_zip_code_id and foi.third_party_id = frm.office_tpid(+)"
		rset, err2 := oraSes.PrepAndQry(qry, userId)
		if err2 != nil {
			log.Println("Oracle error with query=", qry, " ,error=", err2, ", userId=", userId)
			err = err2
		} else {
			for rset.Next() {
				foOfcInfo = FoOfcInfo{Id: rset.Row[10].(ora.OCINum).String(),
					Tpid:    rset.Row[11].(string),
					Name:    rset.Row[12].(string),
					Address: rset.Row[13].(string),
					State:   rset.Row[14].(string),
					City:    rset.Row[15].(string),
					Zip:     rset.Row[16].(string),
					Region:  rset.Row[17].(string),
				}
				foOfcInfoBytes, err3 := json.Marshal(foOfcInfo)
				if err3 != nil {
					log.Println("json.Marshal error:", err3)
					err = err3
				} else {
					var isActive = rset.Row[5].(int64)
					var status string
					if isActive == 1 {
						status = "update"
					} else {
						status = "delete"
					}
					// tmpStr := strconv.FormatInt(isActive, 10) // use base 10 for sanity purpose
					foAgtInfo = FoAgtInfo{Id: rset.Row[0].(ora.OCINum).String(),
						Email:     rset.Row[1].(string),
						Tpid:      rset.Row[2].(string),
						Firstname: rset.Row[3].(string),
						Lastname:  rset.Row[4].(string),
						Status:    status,
						Phone:     rset.Row[6].(string),
						Mobile:    rset.Row[7].(string),
						Fax:       rset.Row[8].(string),
						PhotoUrl:  rset.Row[9].(string),
						Offices:   "[" + string(foOfcInfoBytes) + "]",
					}
				}
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return foAgtInfo, err
}

// get sysdate
func (oracle *Oracle) GetSysdate() (string, error) {
	var sysdateStr string

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	if err == nil {
		qry := "SELECT sysdate from dual"
		rset, err2 := oraSes.PrepAndQry(qry)
		if err2 != nil {
			log.Println("Oracle error with query=", qry, ", error=", err2)
			err = err2
		} else {
			for rset.Next() {
				sysdateStr = rset.Row[0].(time.Time).String()
			}
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}

	return sysdateStr, err
}

// get data from fo_user_info by email_address
// return userWithP
func (oracle *Oracle) GetUserByEmail(email string) (UserWithP, error) {
	var userWithP UserWithP

	oraEnv, oraSrv, oraSes, err := oracle.Connect()
	if err == nil {
		qry := "SELECT fui.first_name, fui.last_name, fui.email_address, fui.day_phone_number, fue.sa_app_code, fai.third_party_id, fui.facebook_id, fui.ip_address, ut_date_to_unix(fui.date_password_expires) date_password_expires, fui.email_is_verified, fui.id de_user_id, fui.password  FROM fo_user_info fui left join fo_user_agent fua on fua.user_id=fui.id left join fo_user_extra fue on fue.user_id=fui.id left join fo_agt_info fai on fai.id=fua.agent_id WHERE lower(fui.email_address) = :email"
		rset, err2 := oraSes.PrepAndQry(qry, strings.ToLower(email))
		if err2 != nil {
			log.Println("Oracle error with query=", qry, ", error=", err2)
			userWithP = UserWithP{}
			err = err2
		} else {
			userWithP = oracle.GetUserFromRset(rset)
		}
		defer oracle.Close(oraEnv, oraSrv, oraSes)
	}
	return userWithP, err
}

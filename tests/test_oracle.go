package main

import (
	"flag"
	"log"
	"net/url"
	"smarteragents/ellimango"
)

var env string
var oraUser string
var oraPass string
var oraSid string

func init() {
	flag.StringVar(&env, "env", "", "Environment (local|staging|production).")
	flag.StringVar(&oraUser, "oraUser", "pde", "Oracle user.")
	flag.StringVar(&oraPass, "oraPass", "p", "Oracle password.")
	flag.StringVar(&oraSid, "oraSid", "de_dev", "Oracle sid.")
	// Parse command-line flags.
	flag.Parse()
}

func main() {
	helper := ellimango.Helper{Env: env}
	oracle := ellimango.Oracle{Env: env,
		OraUser: oraUser,
		OraPass: oraPass,
		OraSid:  oraSid,
	}
	redis := ellimango.Redis{Env: env}
	agent := ellimango.Agent{Oracle: oracle,
		Redis: redis,
	}
	user := ellimango.User{Oracle: oracle,
		Redis: redis,
	}

	foAgtInfo, err := agent.GetByUserId("135237")
	if err != nil {
		log.Println("agent1.GetByUserId error", err)
	} else {
		log.Println("agent 135237", foAgtInfo)
	}

	userWithP, err := user.GetByUserId("135237")
	if err != nil {
		log.Println("user.GetByUserId error", err)
	} else {
		log.Println(userWithP)
	}

	// send map user agent info to edge start
	// response, err := postUserAgentToEdge(&userWithP, &foAgtInfo)
	// send map user agent info to edge end

	// send user info to edge start
	formData := url.Values{}
	formData.Set("login", "pde_user9000")
	formData.Set("password", "usfafgwe")
	formData.Set("action", "mod_pde_user")

	// user found
	if userWithP.DeUserId != "" {
		formData.Set("pde_user_id", userWithP.DeUserId)
		formData.Set("username", userWithP.Email)
		formData.Set("user_password", userWithP.P)
		formData.Set("first_name", userWithP.Firstname)
		formData.Set("last_name", userWithP.Lastname)
		formData.Set("email", userWithP.Email)
		formData.Set("day_phone", userWithP.Phone)
		formData.Set("ip", userWithP.OriginIp)
	} else {
		log.Println("user not found, strange", userWithP)
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
	response, err := helper.Post("http://edge.elliman.com/Sync/PdeUser", formData)
	helper.Debug(formData)
	helper.Debug(err)
	helper.Debug(string(response))
	if `{"status":"success","message":null}` != string(response) {
		helper.Debug("Processing Failed")
	} else {
		helper.Debug("Processing Success")
		if foAgtInfo.Id != "" {
			response, err := postUserAgentToEdge(&userWithP, &foAgtInfo)
			helper.Debug(err)
			helper.Debug(string(response))
		}
	}
	// send user info to edge end
}

// helper - post user agent data to edge elliman
func postUserAgentToEdge(userWithP *ellimango.UserWithP, foAgtInfo *ellimango.FoAgtInfo) ([]byte, error) {
	helper := ellimango.Helper{Env: env}

	formData := url.Values{}
	formData.Set("login", "pde_user9000")
	formData.Set("password", "usfafgwe")
	formData.Set("action", "map_pde_user_agent")

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

	response, err := helper.Post("http://edge.elliman.com/Sync/MapPdeUserAgent", formData)

	return response, err
}

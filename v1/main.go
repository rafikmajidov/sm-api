package main

import (
	"flag"
	"github.com/buaazp/fasthttprouter"
	_ "github.com/pkg/profile"
	"github.com/valyala/fasthttp"
	"log"
	"os"
	_ "runtime"
)

var env string
var requiredAuthtoken string
var oraUser string
var oraPass string
var oraSid string
var redisHost string
var redisDb int
var port string
var rabbitmqUser string
var rabbitmqHost string
var rabbitmqVhost string

func init() {
	flag.StringVar(&env, "env", "local", "Environment (local|staging|production).")
	flag.StringVar(&oraUser, "oraUser", "pde", "Oracle user.")
	flag.StringVar(&oraPass, "oraPass", "p", "Oracle password.")
	flag.StringVar(&oraSid, "oraSid", "de_dev", "Oracle sid.")
	// for redisHost use crestage.dtrleo.0001.use1.cache.amazonaws.com
	flag.StringVar(&redisHost, "redisHost", "", "Redis host.")
	// for redisDb use database 4 or 5 or 6 or 7 or 8
	flag.IntVar(&redisDb, "redisDb", 4, "Redis db.")
	flag.StringVar(&requiredAuthtoken, "authToken", "", "Required authentication token.")
	flag.StringVar(&port, "port", "80", "Web server port.")
	flag.StringVar(&rabbitmqUser, "rabbitmqUser", "", "Rabbitmq user.")
	flag.StringVar(&rabbitmqHost, "rabbitmqHost", "", "Rabbitmq host.")
	flag.StringVar(&rabbitmqVhost, "rabbitmqVhost", "", "Rabbitmq vhost.")
	// Parse command-line flags.
	flag.Parse()
}

func main() {
	// runtime.SetCPUProfileRate(4)
	// profile.MemProfileRate(1024)
	// defer profile.Start(profile.MemProfile, profile.ProfilePath("."), profile.NoShutdownHook).Stop()
	if env != "local" && env != "staging" && env != "production" {
		log.Printf("Wrong env `%s`.", env)
		os.Exit(1)
	}

	if requiredAuthtoken == "" {
		log.Println(PLS_SP_AUTH)
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

	router := fasthttprouter.New()
	// GET /health/:authtoken
	// GET /dualagents - Get all Dual Agents
	// POST /user/login - Authenticate user
	// POST /user/reset/{de_user_id} - Reset the expires on to current date plus 120 days
	// POST /user/create - Create New User
	// PUT /user/{de_user_id} - Update Existing User
	// GET /user/{de_user_id} - Get Existing User Information
	// PUT /listings/{de_user_id} - Add Listing to User Folder
	// GET /listings/{de_user_id}/ - Get All User Saved Listings
	router.GET("/v1/health/:authtoken", Handle(GetHealth, Auth, SetAuthtokenFromUrl))
	router.GET("/v1/dualagents", Handle(GetDualAgents, Auth))
	router.POST("/v1/user/login", Handle(AuthenticateUser, ValidateLogin, Auth))
	router.POST("/v1/user/reset/:userId", Handle(ResetUserExpiration, ValidateUserId, Auth))
	router.POST("/v1/user/verify/:userId", Handle(VerifyUser, ValidateUserId, Auth))
	router.PUT("/v1/listings/:userId", Handle(AddUserListing, ValidateAddUserListing, ValidateUserId, Auth))
	router.DELETE("/v1/listings/:userId", Handle(DeleteUserListing, ValidateDeleteUserListing, ValidateUserId, Auth))
	router.GET("/v1/listings/:userId", Handle(GetUserListings, ValidateUserId, Auth))
	router.GET("/v1/user/:userId", Handle(GetUser, ValidateUserId, Auth))
	router.POST("/v1/user/create", Handle(CreateUser, ValidateEmailForCreate, ValidateDeAgentId, ValidateCreateUpdate, Auth))
	router.PUT("/v1/user/:userId", Handle(UpdateUser, ValidateDeAgentId, ValidateCreateUpdate, ValidateUserId, Auth))
	router.GET("/v1/useremail/:email", Handle(GetUserByEmail, Auth))
	log.Println("Starting HTTP server on :" + port)
	log.Fatal(fasthttp.ListenAndServe(":"+port, router.Handler))
}

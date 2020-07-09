package main

import (
	"flag"
	"log"
	"os"
	"smarteragents/ellimango"
)

var env string
var rabbitmqUser string
var rabbitmqHost string
var rabbitmqVhost string

func init() {
	flag.StringVar(&env, "env", "", "Environment (local|staging|production).")
	flag.StringVar(&rabbitmqUser, "rabbitmqUser", "", "Rabbitmq user.")
	flag.StringVar(&rabbitmqHost, "rabbitmqHost", "", "Rabbitmq host.")
	flag.StringVar(&rabbitmqVhost, "rabbitmqVhost", "", "Rabbitmq vhost.")
	// Parse command-line flags.
	flag.Parse()
}

func main() {
	if env != "local" && env != "staging" && env != "production" {
		log.Printf("Wrong env `%s`.", env)
		os.Exit(1)
	}
	if rabbitmqUser == "" {
		log.Println("Please specify rabbitmq user")
		os.Exit(1)
	}

	if rabbitmqHost == "" {
		log.Println("Please specify rabbitmq host")
		os.Exit(1)
	}

	if rabbitmqVhost == "" {
		log.Println("Please specify rabbitmq vhost")
		os.Exit(1)
	}

	rabbitmq := ellimango.Rabbitmq{Env: env,
		RabbitmqUser:  rabbitmqUser,
		RabbitmqHost:  rabbitmqHost,
		RabbitmqVhost: rabbitmqVhost,
	}

	helper := ellimango.Helper{Env: env}
	rabbitmqStatus, err := helper.GetRabbitmqStatus(rabbitmq)
	log.Println(rabbitmqStatus)
	log.Println(err)
}

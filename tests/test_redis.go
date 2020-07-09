package main

import (
	"flag"
	//"github.com/garyburd/redigo/redis"
        "github.com/gomodule/redigo/redis"
	"log"
	"os"
	"time"
)

var (
	redisHost = flag.String("redisHost", "", "Redis host.")
	redisDb   = flag.Int("redisDb", 4, "Redis db.")
)

func main() {
	flag.Parse()
	if *redisHost == "" {
		log.Println("Please specify redis host")
		os.Exit(1)
	}
	log.Printf("redishost %s, redisDb %d", *redisHost, *redisDb)
	c, err := redis.DialURL(*redisHost, redis.DialConnectTimeout(time.Duration(3)*time.Second))
	if err != nil {
		log.Print("redis conn error")
		log.Print(err)
		// handle connection error
	} else {
		log.Print("redis conn")
		log.Print(c)
		res, err := c.Do("SELECT", *redisDb)
		if err != nil {
			log.Print("redis select db error")
			log.Print(err)

		} else {
			log.Print("redis select")
			log.Print(res)
			/*
				res, err := c.Do("SADD", "dual_agents", "primary_tpid1:secondary_tpid1")
				if err != nil {
					log.Print("redis sadd error")
					log.Print(err)
				} else {
					log.Print("redis sadd")
					log.Print(res)
					res, err := redis.Strings(c.Do("SMEMBERS", "dual_agents"))
					if err != nil {
						log.Print("redis smembers error")
						log.Print(err)
					} else {
						log.Print("redis smembers")
						log.Print(res)
						log.Println(len(res))
						dualAgents := make(map[string]map[int]string)
						var sIdLen int
						if len(res) != 0 {
							for key, value := range res {
								log.Println(key, value)
								if value != "" {
									tmpArr := strings.Split(value, ":")
									log.Println(tmpArr)
									if len(tmpArr) == 2 {
										pId := fmt.Sprintf("%s", tmpArr[0])
										sId := fmt.Sprintf("%s", tmpArr[1])

										if dualAgents[pId] == nil {
											dualAgents[pId] = make(map[int]string)
										}

										sIdLen = len(dualAgents[pId])
										dualAgents[pId][sIdLen] = sId
									}
								}
							}
							log.Println(dualAgents)
						}
					}
				}
			*/
		}
		defer c.Close()
	}

}

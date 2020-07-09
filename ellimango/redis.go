package ellimango

import (
	"encoding/json"
	_ "errors"
	//"github.com/garyburd/redigo/redis"
        "github.com/gomodule/redigo/redis"
	"log"
	"strings"
	"time"
)

type Redis struct {
	Env       string
	RedisHost string
	RedisDb   int
}

// connect to redis
func (r *Redis) Connect() (redis.Conn, error) {
	helper := Helper{Env: r.Env}

	c, err := redis.DialURL(r.RedisHost, redis.DialConnectTimeout(time.Duration(2)*time.Second), redis.DialKeepAlive(time.Duration(3)*time.Minute))
	// handle connection error
	if err != nil {
		log.Println("Redis connect error", err)
		//go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Redis connect error %v", err), "Redis connect error")
		return nil, err
	} else {
		res, err := c.Do("SELECT", r.RedisDb)
		if err != nil {
			log.Println("Redis select error", err)
			//go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Redis select error %v", err), "Redis select error")
			return nil, err
		} else {
			helper.Debug("Redis select", res)
			return c, nil
		}
	}
}

// close redis connection
func (r *Redis) Close(c redis.Conn) error {
	helper := Helper{Env: r.Env}

	err := c.Close()
	if err != nil {
		log.Println("Redis close error", err)
	} else {
		helper.Debug("Redis close connect ok")
	}
	return err
}

// get all dual agents
func (r *Redis) GetDualAgents() map[string]map[int]string {
	dualAgents := make(map[string]map[int]string)

	c, err := r.Connect()
	// no connect/select error
	if err == nil {
		res, err := redis.Strings(c.Do("SMEMBERS", "dual_agents"))
		// no smembers command error
		if err == nil {
			// not empty
			if len(res) != 0 {
				var sIdLen int
				for _, value := range res {
					if value != "" {
						tmpArr := strings.Split(value, ":")

						if len(tmpArr) == 2 {
							pId := tmpArr[0]
							sId := tmpArr[1]

							if dualAgents[pId] == nil {
								dualAgents[pId] = make(map[int]string)
							}

							sIdLen = len(dualAgents[pId])
							dualAgents[pId][sIdLen] = sId
						}
					}
				}
			}
		} else {
			log.Println("Redis smembers dual_agents error", err)
		}
		defer r.Close(c)
	}

	return dualAgents
}

// save in the set dual_agents primary_agent_third_party_id:secondary_agent_third_party_id
func (r *Redis) SaveDualAgent(c redis.Conn, pIdSid string) (interface{}, error) {
	res, err := c.Do("SADD", "dual_agents", pIdSid)
	return res, err
}

// get user by id
func (r *Redis) GetUserByUserId(userId string) (UserWithP, error) {
	helper := Helper{Env: r.Env}
	var userWithP UserWithP

	c, err := r.Connect()
	// no connect and select error
	if err == nil {
		reply, err := c.Do("GET", "user:"+userId)
		if err == nil {
			helper.Debug("Redis GetUserByUserId(" + userId + ") was ok")
			// user found in redis by userId
			if reply != nil {
				err = json.Unmarshal(reply.([]byte), &userWithP)
				if err != nil {
					log.Println("Redis GetUserByUserId("+userId+") unmarshall error", err)
					userWithP = UserWithP{}
				}
			} else {
				helper.Debug("Redis GetUserByUserId(" + userId + ") was not found in redis")
			}
		} else {
			log.Println("Redis GetUserByUserId("+userId+") error=", err)
		}
		defer r.Close(c)
	}

	return userWithP, err
}

// get user by email and password
func (r *Redis) GetUserByEmailAndPassword(email string, password string) (UserWithP, error) {
	helper := Helper{Env: r.Env}
	var userWithP UserWithP

	c, err := r.Connect()
	// no connect and select error
	if err == nil {
		reply, err := c.Do("GET", "user:"+email)
		if err == nil {
			helper.Debug("Redis GetUserByEmailAndPassword(" + email + ") was ok")
			// user found in redis by email
			if reply != nil {
				err = json.Unmarshal(reply.([]byte), &userWithP)
				if err != nil {
					log.Println("Redis GetUserByEmailAndPassword("+email+") unmarshall error", err)
					userWithP = UserWithP{}
				} else {
					// passwords do not match, set userWithP to empty struct
					if userWithP.P != password {
						log.Println("Passwords do not match", userWithP.P, password)
						userWithP = UserWithP{}
					}
				}
			} else {
				helper.Debug("Redis GetUserByEmailAndPassword(" + email + ") was not found in redis")
			}
		} else {
			log.Println("Redis GetUserByEmailAndPassword("+email+") error=", err)
		}
		defer r.Close(c)
	}

	return userWithP, err
}

// connect to redis
// save in redis json
// close redis connection
func (r *Redis) SaveJson(key string, js []byte) (bool, error) {
	helper := Helper{Env: r.Env}
	c, err := r.Connect()

	// connect or select error
	if err != nil {
		return false, err
		// no connect and select error
	} else {
		reply, err := c.Do("SET", key, js)
		// set command error
		if err != nil {
			log.Println("Redis SaveJson error", err)
			return false, err
			// set command no error
		} else {
			helper.Debug("Redis SaveJson "+string(js)+" reply", reply)
		}
		defer r.Close(c)
	}

	return true, nil
}

// get listings saved by user
func (r *Redis) GetUserListings(userId string) SavedListings {
	helper := Helper{Env: r.Env}
	var userListings SavedListings

	c, err := r.Connect()
	// no connect and select error
	if err == nil {
		reply, err := c.Do("GET", "userListings:"+userId)
		if err == nil {
			helper.Debug("Redis GetUserListings(" + userId + ") was ok")
			// userListings found in redis by userId
			if reply != nil {
				err = json.Unmarshal(reply.([]byte), &userListings)
				if err != nil {
					log.Println("Redis GetUserListings("+userId+") unmarshall error", err)
					userListings = SavedListings{}
				}
			} else {
				helper.Debug("Redis GetUserListings(" + userId + ") was not found in redis")
			}
		} else {
			log.Println("Redis GetUserListings("+userId+") error=", err)
		}
		defer r.Close(c)
	}

	return userListings
}

// Ping
func (r *Redis) Ping() (string, error) {
	var pingResponse string

	c, err := r.Connect()
	// no connect and select error
	if err == nil {
		_, err2 := c.Do("PING")
		if err2 == nil {
			pingResponse = "PONG"
		} else {
			log.Println("Redis ping err", err2)
			err = err2
		}
		defer r.Close(c)
	}

	return pingResponse, err
}

// go routine to save user in redis
func (r *Redis) WorkerSaveUser(usersWithP chan UserWithP) {
	h := Helper{Env: r.Env}
	h.Debug("redisWorkerSaveUser() before sending to redis  time.Now " + time.Now().String())

	for {
		userWithP, more := <-usersWithP
		if more {
			js, err := json.Marshal(userWithP)
			if err == nil {
				// Save JSON blob to Redis
				res, err := r.SaveJson("user:"+userWithP.Email, js)
				h.Debug("After saving userWithPassword by email in redis, err=", err, "res=", res)
				res, err = r.SaveJson("user:"+userWithP.DeUserId, js)
				h.Debug("After saving userWithPassword by userId in redis, err=", err, "res=", res)
				h.Debug("redisWorkerSaveUser() after sending to redis one usersWithPassword time.Now " + time.Now().String())
			} else {
				log.Println("redisWorkerSaveUser json marshall error", err)
			}
		} else {
			h.Debug("Received all usersWithPassword")
			h.Debug("redisWorkerSaveUser() after sending to redis all usersWithPassword time.Now " + time.Now().String())
			return
		}
	}
}

// go routine to save user listings in redis
func (r *Redis) WorkerSaveUserListings(userListingsChan chan SavedListings) {
	h := Helper{Env: r.Env}
	h.Debug("redisWorkerSaveUserListings() before sending to redis time.Now " + time.Now().String())

	for {
		userListings, more := <-userListingsChan
		if more {
			js, err := json.Marshal(userListings)
			if err == nil {
				// Save JSON blob to Redis
				res, err := r.SaveJson("userListings:"+userListings.DeUserId, js)
				h.Debug("redisWorkerSaveUserListings() after sending to redis one userListings time.Now " + time.Now().String())
				h.Debug("After saving userListings by userId in redis, err=", err, "res=", res)
			} else {
				log.Println("redisWorkerSaveUserListings json marshall error", err)
			}
		} else {
			h.Debug("Received all userListings")
			h.Debug("redisWorkerSaveUserListings() after sending to redis all userListings time.Now " + time.Now().String())
			return
		}
	}
}

// go routine to save all dual agents in redis
func (r *Redis) WorkerSaveAllDualAgents(pIdSids []string) {
	h := Helper{Env: r.Env}
	h.Debug("redisWorkerSaveAllDualAgents() before sending to redis  time.Now " + time.Now().String())
	c, err := r.Connect()

	// no connection and select errors
	if err == nil {
		for index, pIdSid := range pIdSids {
			res, err := r.SaveDualAgent(c, pIdSid)
			h.Debug("index", index)
			h.Debug("Redis saveDualAgent pIdSid="+pIdSid+", res=", res, "err=", err)
		}
		defer r.Close(c)
	}
	h.Debug("redisWorkerSaveAllDualAgents() after sending to redis  time.Now " + time.Now().String())
}

// get user by email
func (r *Redis) GetUserByEmail(email string) (UserWithP, error) {
	helper := Helper{Env: r.Env}
	var userWithP UserWithP

	c, err := r.Connect()
	// no connect and select error
	if err == nil {
		reply, err := c.Do("GET", "user:"+email)
		if err == nil {
			helper.Debug("Redis GetUserByEmail(" + email + ") was ok")
			// user found in redis by email
			if reply != nil {
				err = json.Unmarshal(reply.([]byte), &userWithP)
				if err != nil {
					log.Println("Redis GetUserByEmail("+email+") unmarshall error", err)
					userWithP = UserWithP{}
				}
			} else {
				helper.Debug("Redis GetUserByEmail(" + email + ") was not found in redis")
			}
		} else {
			log.Println("Redis GetUserByEmail("+email+") error=", err)
		}
		defer r.Close(c)
	}

	return userWithP, err
}

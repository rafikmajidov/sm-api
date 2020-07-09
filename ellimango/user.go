package ellimango

import (
	"log"
)

type User struct {
	Oracle   Oracle
	Redis    Redis
	Rabbitmq Rabbitmq
}

// get user by userId
func (u *User) GetByUserId(userId string) (UserWithP, error) {
	h := Helper{Env: u.Oracle.Env}
	// get user from redis
	userWithP, err := u.Redis.GetUserByUserId(userId)
	// user not found in redis or there was an error
	if userWithP.DeUserId == "" || err != nil {
		h.Debug("No redis user by userId, let's get from oracle")
		// get user from oracle
		userWithP, err = u.Oracle.GetUserByUserId(userId)
		// user found in oracle
		// save user in redis
		if userWithP.DeUserId != "" && err == nil {
			usersWithP := make(chan UserWithP, 1)
			usersWithP <- userWithP
			go u.Redis.WorkerSaveUser(usersWithP)
		}
	}
	return userWithP, err
}

// get user by email and password
func (u *User) GetByEmailAndPassword(email string, password string) (UserWithP, error) {
	h := Helper{Env: u.Oracle.Env}
	// get user from redis
	userWithP, err := u.Redis.GetUserByEmailAndPassword(email, password)
	// user not found in redis or there was an error
	if userWithP.DeUserId == "" || err != nil {
		h.Debug("No redis user by email and password, let's get from oracle")
		// get user from oracle
		userWithP, err = u.Oracle.GetUserByEmailAndPassword(email, password)
		// user found in oracle
		// save user in redis
		if userWithP.DeUserId != "" && err == nil {
			usersWithP := make(chan UserWithP, 1)
			usersWithP <- userWithP
			go u.Redis.WorkerSaveUser(usersWithP)
		}
	}
	return userWithP, err
}

// get user id by email
func (u *User) GetUserIdByEmail(email string) (string, error) {
	userId, err := u.Oracle.GetUserIdByEmail(email)
	return userId, err
}

// create new user
// send info to edge
func (u *User) Create(firstname string, lastname string, email string, phone string, password string, saAppCode string, deAgentId string, fbId string, originIp string, agentId uint64) (UserWithP, string) {
	userWithP, errorMessage := u.Oracle.CreateUser(firstname, lastname, email, phone, password, saAppCode, deAgentId, fbId, originIp, agentId)
	// save user in redis, save data in rabbitmq
	if errorMessage == "" && userWithP.DeUserId != "" {
		// save user in redis
		usersWithP := make(chan UserWithP, 1)
		usersWithP <- userWithP
		go u.Redis.WorkerSaveUser(usersWithP)

		// send to edge user
		// if agent is present, send to edge user agent
		task1 := RabbitmqTask{SyncAction: MOD_PDE_USER,
			DeUserId:     userWithP.DeUserId,
			OldAgentTpid: "",
			NewAgentId:   agentId,
		}
		go u.Rabbitmq.WorkerSaveTask(&task1)

		// send to edge user agent
		// only if user has agent
		// if agentId > 0 {
		// 	task2 := RabbitmqTask{SyncAction: MAP_PDE_USER_AGENT,
		// 		DeUserId:     userWithP.DeUserId,
		// 		OldAgentTpid: "",
		// 		NewAgentId:   agentId,
		// 	}
		// 	go u.Rabbitmq.WorkerSaveTask(&task2)
		// }
	}
	return userWithP, errorMessage
}

// update existing user
// send info to edge
func (u *User) Update(firstname string, lastname string, email string, phone string, password string, saAppCode string, deAgentId string, fbId string, originIp string, agentId uint64, userWithP UserWithP) (UserWithP, string) {
	updatedUserWithP := userWithP
	errorMessage := ""

	// if some fields changed, update user
	if userWithP.Firstname != firstname || userWithP.Lastname != lastname || userWithP.Email != email || userWithP.Phone != phone || userWithP.SaAppCode != saAppCode || userWithP.DeAgentId != deAgentId || userWithP.FbId != fbId || userWithP.OriginIp != originIp || userWithP.P != password {

		updatedUserWithP, errorMessage := u.Oracle.UpdateUser(firstname, lastname, email, phone, password, saAppCode, deAgentId, fbId, originIp, agentId, userWithP)

		if errorMessage == "" && updatedUserWithP.DeUserId != "" {
			// save user in redis
			usersWithP := make(chan UserWithP, 1)
			usersWithP <- updatedUserWithP
			go u.Redis.WorkerSaveUser(usersWithP)
		}
		// send to edge
		// agent has changed
		if userWithP.DeAgentId != deAgentId {
			var agentAction = ""
			var oldAgentTpid = ""
			var newAgentId uint64
			// remove agent from user
			if userWithP.DeAgentId != "" && deAgentId == "" {
				// old agent third party id is known : userWithP.DeAgentId
				agentAction = "delete-agent"
				oldAgentTpid = userWithP.DeAgentId
				newAgentId = 0

				// assign new agent to user
			} else if userWithP.DeAgentId == "" && deAgentId != "" && agentId > 0 {
				// old agent is null, new agent id is known : agentId
				agentAction = "add-agent"
				oldAgentTpid = ""
				newAgentId = agentId

				// update agent of user
			} else if userWithP.DeAgentId != "" && deAgentId != "" && agentId > 0 {
				// old agent third party id is known: userWithP.DeAgentId
				// new agent id is known: agentId
				agentAction = "update-agent"
				oldAgentTpid = userWithP.DeAgentId
				newAgentId = agentId
			}
			if agentAction != "" {
				// sync user agent to edge
				task := RabbitmqTask{SyncAction: MAP_PDE_USER_AGENT,
					DeUserId:     userWithP.DeUserId,
					OldAgentTpid: oldAgentTpid,
					NewAgentId:   newAgentId,
				}

				go u.Rabbitmq.WorkerSaveTask(&task)
			}
		}
	} else {
		log.Println("user update data is the same")
	}

	return updatedUserWithP, errorMessage
}

// update existing user password expiration
func (u *User) UpdateUserExpiration(userId string) (UserWithP, string) {
	userWithP, errorMessage := u.Oracle.UpdateUserExpiration(userId)
	// save user in redis, save data in rabbitmq
	if errorMessage == "" && userWithP.DeUserId != "" {
		// save user in redis
		usersWithP := make(chan UserWithP, 1)
		usersWithP <- userWithP
		go u.Redis.WorkerSaveUser(usersWithP)
	}
	return userWithP, errorMessage
}

// update user  in redis
// get user from oracle
// save user in redis
func (u *User) RedisUpdate(userId string) string {
	h := Helper{Env: u.Oracle.Env}
	var errorMessage string
	// get user from oracle
	userWithP, err := u.Oracle.GetUserByUserId(userId)
	h.Debug("user RedisUpdate", userWithP, err)
	// user found in oracle
	// save user in redis
	if userWithP.DeUserId != "" && err == nil {
		usersWithP := make(chan UserWithP, 1)
		usersWithP <- userWithP
		go u.Redis.WorkerSaveUser(usersWithP)
		errorMessage = ""
	} else {
		if err != nil {
			log.Println("oracle GetUserByUserId err", err)
		}
		if userWithP.DeUserId == "" {
			log.Println("User not found in oracle by userId", userId)
		}

		errorMessage = "User not updated in redis"
	}

	return errorMessage
}

func (u *User) RabbitmqMembershipVerificationEmail(userId string) {
	// send user info to elliman, so that elliman sends membership verification email to user
	task := RabbitmqTask{SyncAction: MEMB_VER_EMAIL,
		DeUserId: userId,
	}

	go u.Rabbitmq.WorkerSaveTask(&task)
}

// get user by email
func (u *User) GetByEmail(email string) (UserWithP, error) {
	h := Helper{Env: u.Oracle.Env}
	// get user from redis
	userWithP, err := u.Redis.GetUserByEmail(email)
	// user not found in redis or there was an error
	if userWithP.DeUserId == "" || err != nil {
		h.Debug("No redis user by email, let's get from oracle")
		// get user from oracle
		userWithP, err = u.Oracle.GetUserByEmail(email)
		// user found in oracle
		// save user in redis
		if userWithP.DeUserId != "" && err == nil {
			usersWithP := make(chan UserWithP, 1)
			usersWithP <- userWithP
			go u.Redis.WorkerSaveUser(usersWithP)
		}
	}
	return userWithP, err
}

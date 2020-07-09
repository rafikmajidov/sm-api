package ellimango

type UserListing struct {
	Oracle   Oracle
	Redis    Redis
	Rabbitmq Rabbitmq
}

// get user all saved listings
func (ul *UserListing) GetAll(userId string) (SavedListings, string) {
	h := Helper{Env: ul.Oracle.Env}
	var userListings SavedListings
	var errorMessage string

	// get user listings from redis
	userListings = ul.Redis.GetUserListings(userId)
	// user listings not found in redis
	if userListings.DeUserId == "" {
		h.Debug("No redis userListings by userId, let's get from oracle")
		// get user listings from oracle
		userListings, errorMessage = ul.Oracle.GetUserListings(userId)
		// user listings found in oracle
		// save them in redis
		if errorMessage == "" && userListings.DeUserId != "" {
			// save user listings in redis
			userListingsChan := make(chan SavedListings, 1)
			userListingsChan <- userListings
			go ul.Redis.WorkerSaveUserListings(userListingsChan)
		}
	}
	return userListings, errorMessage
}

// add user listing
func (ul *UserListing) Add(userId string, folderId string, listingId string) (SavedListings, string) {
	var userListings SavedListings
	// folder must exist at this moment
	// insert user id, folder id, listing id into prof_saved_listing
	errorMessage := ul.Oracle.CreateUserSavedListing(userId, folderId, listingId)
	// listing was saved
	if errorMessage == "" {
		// sync user listing to edge
		task := RabbitmqTask{SyncAction: ADD_SVD_APT,
			ListingId: listingId,
			FolderId:  folderId,
			DeUserId:  userId,
		}
		go ul.Rabbitmq.WorkerSaveTask(&task)

		// get user listings from oracle
		userListings, errorMessage = ul.Oracle.GetUserListings(userId)
		// user listings found in oracle
		// save them in redis
		if errorMessage == "" && userListings.DeUserId != "" {
			// save user listings in redis
			userListingsChan := make(chan SavedListings, 1)
			userListingsChan <- userListings
			go ul.Redis.WorkerSaveUserListings(userListingsChan)
		}
	}

	return userListings, errorMessage
}

// find user listing by user id, listing id and folder id
func (ul *UserListing) Find(userId string, folderId string, listingId string) bool {
	found := ul.Oracle.FindUserSavedListing(userId, folderId, listingId)
	return found
}

// delete user listing by user id, listing id and folder id
func (ul *UserListing) Delete(userId string, folderId string, listingId string) (SavedListings, string) {
	var userListings SavedListings

	errorMessage := ul.Oracle.DeleteUserSavedListing(userId, folderId, listingId)
	// listing was deleted
	if errorMessage == "" {
		// sync user listing to edge
		task := RabbitmqTask{SyncAction: DEL_SVD_APT,
			ListingId: listingId,
			FolderId:  folderId,
			DeUserId:  userId,
		}
		go ul.Rabbitmq.WorkerSaveTask(&task)

		// get user listings from oracle
		userListings, errorMessage = ul.Oracle.GetUserListings(userId)
		// user listings found in oracle
		// save them in redis
		if errorMessage == "" && userListings.DeUserId != "" {
			// save user listings in redis
			userListingsChan := make(chan SavedListings, 1)
			userListingsChan <- userListings
			go ul.Redis.WorkerSaveUserListings(userListingsChan)
		}
	}

	return userListings, errorMessage
}

// update all user saved listings in redis
// get all user listings from oracle
// save them in redis
func (ul *UserListing) RedisUpdate(userId string) string {
	h := Helper{Env: ul.Oracle.Env}
	// get user listings from oracle
	userListings, errorMessage := ul.Oracle.GetUserListings(userId)
	h.Debug("userlisting RedisUpdate", userListings, errorMessage)
	// user listings found in oracle
	// save them in redis
	if errorMessage == "" && userListings.DeUserId != "" {
		// save user listings in redis
		userListingsChan := make(chan SavedListings, 1)
		userListingsChan <- userListings
		go ul.Redis.WorkerSaveUserListings(userListingsChan)
	}

	return errorMessage
}

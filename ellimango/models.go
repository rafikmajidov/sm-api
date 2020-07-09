package ellimango

// used for json response
type Response struct {
	Reason string `json:"reason"`
	Version string `json:"version"`
}

// slice
type SecondaryAgentIds []string

// used for json response in GetDualAgents handler
type DualAgent struct {
	PrimaryAgentId   string            `json:"primary_agent_id"`
	SecondaryAgentId SecondaryAgentIds `json:"secondary_agent_id"`
}

// used for json response in GetUser handler
type UserWithP struct {
	DeUserId      string `json:"de_user_id"`
	Firstname     string `json:"firstname"`
	Lastname      string `json:"lastname"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	SaAppCode     string `json:"sa_app_code"`
	DeAgentId     string `json:"de_agent_id"`
	FbId          string `json:"fb_id"`
	OriginIp      string `json:"origin_ip"`
	ExpiresOn     string `json:"expires_on"`
	EmailVerified int64  `json:"email_verified"`
	P             string `json:"p,omitempty"`
	ExpiresOnTs   int64  `json:"expires_on_ts,omitempty"`
}

type UserShort struct {
	Reason   string `json:"reason,omitempty"`
	DeUserId string `json:"de_user_id"`
}

type SavedListing struct {
	ListingTpid    string `json:"listing_id"`
	AffiliateId    string `json:"listing_aff_id"`
	SubAffiliateId string `json:"listing_sub_aff_id"`
}

type SavedFolder struct {
	Id       uint64         `json:"id,omitempty"`
	Name     string         `json:"name"`
	SourceId uint64         `json:"source_id"`
	Listings []SavedListing `json:"listings"`
}

// used for json response in GetUser handler
type SavedListings struct {
	DeUserId string        `json:"de_user_id"`
	Folders  []SavedFolder `json:"folders"`
}

// used to send data to rabbitmq
type RabbitmqTask struct {
	SyncAction   string `json:"sync_action,omitempty"`
	DeUserId     string `json:"de_user_id,omitempty"`
	OldAgentTpid string `json:"old_agent_tpid,omitempty"`
	NewAgentId   uint64 `json:"new_agent_id"`
	ListingId    string `json:"listing_id,omitempty"`
	FolderId     string `json:"folder_id,omitempty"`
}

type FoAgtInfo struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	Tpid      string `json:"third_party_id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Status    string `json:"status"`
	Phone     string `json:"phone"`
	Mobile    string `json:"mobile"`
	Fax       string `json:"fax"`
	PhotoUrl  string `json:"photo_url"`
	Offices   string `json:"offices"`
}

type FoOfcInfo struct {
	Id      string `json:"id"`
	Tpid    string `json:"third_party_id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	State   string `json:"state"`
	City    string `json:"city"`
	Zip     string `json:"zip"`
	Region  string `json:"region"`
}

type SolrParams struct {
	Q  string `json:"q"`
	Fl string `json:"fl"`
	Wt string `json:"wt"`
}

type SolrListing struct {
	Tpid             string  `json:"third_party_id"`
	BldgTypeId       uint8   `json:"building_type_id"`
	TransTypeId      uint8   `json:"transaction_type_id"`
	CurrentPrice     string  `json:"current_price"`
	RegionId         string  `json:"region_id"`
	NeighborhoodId   string  `json:"neighborhood_id"`
	NumBedrooms      float32 `json:"num_bedrooms"`
	Area             float32 `json:"area"`
	StatusId         uint8   `json:"status_id"`
	AgencyName       string  `json:"agency_name"`
	LclRegionId      string  `json:"lcl_region_id"`
	NeighborhoodName string  `json:"neighborhood_name"`
	Longitude        string  `json:"longitude"`
	Latitude         string  `json:"latitude"`
	DisplayName      string  `json:"display_name"`
	Url              string  `json:"url"`
	FullTimeDoorman  uint8   `json:"full_time_doorman"`
	PartTimeDoorman  uint8   `json:"part_time_doorman"`
	BuildingId       uint64  `json:"building_id"`
	AffiliateId      uint8   `json:"affiliate_id"`
	DisplayAttribute string  `json:"display_attribute"`
}

type SolrResponseHeader struct {
	Status uint       `json:"status"`
	Qtime  uint       `json:"QTime"`
	Params SolrParams `json:"params"`
}

type SolrListingResponseInner struct {
	NumFound uint          `json:"numFound"`
	Start    uint          `json:"start"`
	Docs     []SolrListing `json:"docs"`
}

type SolrListingResponse struct {
	ResponseHeader SolrResponseHeader       `json:"responseHeader"`
	ResponseInner  SolrListingResponseInner `json:"response"`
}

type RabbitmqStatus struct {
	Name     string `json:"name"`
	Messages uint64 `json:"messages"`
	Memory   uint64 `json:"memory"`
	Node     string `json:"node"`
}

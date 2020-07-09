package ellimango

type Listing struct {
	Oracle Oracle
	Redis  Redis
}

// get listing id by third party id and db affiliate id
func (listing *Listing) GetListingIdByTpidAndAffId(listingTpid string, dbAffiliateId string) (string, string) {
	listingId, errorMessage := listing.Oracle.GetListingIdByTpidAndAffId(listingTpid, dbAffiliateId)
	return listingId, errorMessage
}

package main

import (
	"flag"
	"fmt"
	"log"
	_ "net/url"
	"os"
	"smarteragents/ellimango"
)

// http://dev.legacy.cityrealty.com/api-pde/ - staging
// http://local.legacy.cityrealty.com/api-pde/index2.php - local legacy cityrealty
// http://local.edge.com/sync/AddSavedApartment?password=usfafgwe&pde_user_id=12 - local edge
var env string
var postUrl string
var listingId string

func init() {
	flag.StringVar(&env, "env", "", "Environment (local|staging|production).")
	flag.StringVar(&postUrl, "postUrl", "", "Url to post data.")
	flag.StringVar(&listingId, "listingId", "", "Listing Id to get data from solr")
	// Parse command-line flags.
	flag.Parse()
}

func main() {
	if env != "local" && env != "staging" && env != "production" {
		log.Printf("Wrong env `%s`.", env)
		os.Exit(1)
	}

	if postUrl == "" {
		log.Println("Please specify url where to post data")
		os.Exit(1)
	}

	if listingId == "" {
		log.Println("Please specify listingId")
		os.Exit(1)
	}

	// formData := url.Values{}
	// formData.Set("login", "pde_user9000")
	// formData.Set("password", "usfafgwe")

	helper := ellimango.Helper{Env: env}

	// body, err := helper.Post(postUrl, formData)
	// log.Println(err)
	// log.Println(string(body))

	// second test
	solrListing, err := helper.GetSolrListingDataForAddSavedApartment(listingId)
	log.Println(err)
	fmt.Printf("solrListing=%+v\n", solrListing)
}

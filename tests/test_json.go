package main

import (
	"encoding/json"
	"fmt"
	"log"
)

func main() {

	type Office struct {
		Id      string `json:"id"`
		Tpid    string `json:"third_party_id"`
		Name    string `json:"name"`
		Address string `json:"address"`
		State   string `json:"state"`
		City    string `json:"city"`
		Zip     string `json:"zip"`
		Region  string `json:"region"`
	}

	type Agt struct {
		Id        string `json:"id"`
		Email     string `json:"email"`
		Tpid      string `json:"third_party_id"`
		Firstname string `json:"firstname"`
		Lastname  string `json:"lastname"`
		Status    string `json:"status"`
		Mobile    string `json:"mobile"`
		Phone     string `json:"phone"`
		Fax       string `json:"fax"`
		PhotoUrl  string `json:"photo_url"`
		Offices   string `json:"offices"`
	}

	office := Office{
		Id:      "idtest",
		Tpid:    "tpidtest",
		Name:    "name",
		Address: "add1",
		State:   "st1",
		City:    "city1",
		Zip:     "zip1",
		Region:  "r1",
	}

	b, err := json.Marshal(office)
	if err != nil {
		log.Println("error:", err)
	} else {

		agt := Agt{
			Id:        "id",
			Email:     "email",
			Tpid:      "third_party_id",
			Firstname: "firstname",
			Lastname:  "lastname",
			Status:    "status",
			Mobile:    "mobile",
			Phone:     "phone",
			Fax:       "fax",
			PhotoUrl:  "photo_url",
			Offices:   string(b),
		}
		fmt.Printf("%v %T\n", agt, agt)
	}

	// formData := url.Values{}
	// formData.Set("login", "pde_user9000")
	// formData.Set("password", "usfafgwe")

	// helper := ellimango.Helper{Env: env}

	// body, err := helper.Post(postUrl, formData)
	// log.Println(err)
	// log.Println(string(body))

}

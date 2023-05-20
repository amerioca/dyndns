package dyndns

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/oze4/godaddygo"
)

var IP_PROVIDER = "https://v4.ident.me/"

func getOwnIPv4() (string, error) {
	resp, err := http.Get(IP_PROVIDER)
	if err != nil {
		fmt.Printf("getOwnIPv4 ERR")
		return "", err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String(), nil
}

func getDomainIPv4() (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/A/%s", DOMAIN, SUBDOMAIN), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", GODADDY_KEY, GODADDY_SECRET))
	c := new(http.Client)
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	in := make([]struct {
		Data string `json:"data"`
	}, 1)
	json.NewDecoder(resp.Body).Decode(&in)
	return in[0].Data, nil
}

func putNewIP(ip string) error {
	// Connect to production Gateway
	api, _ := godaddygo.NewProduction(GODADDY_KEY, GODADDY_SECRET)
	// Target version 1 of the production GoDaddy Gateway
	prodv1 := api.V1()
	// Set our domain
	domain := prodv1.Domain(DOMAIN)
	// Target `records` for this domain
	recs := domain.Records()
	// Update existing record
	newrecord := godaddygo.Record{
		Data: ip,
	}

	if err := recs.ReplaceByTypeAndName(context.Background(), godaddygo.RecordTypeA, SUBDOMAIN, newrecord); err != nil {
		return fmt.Errorf("error in TestRecordReplaceByTypeAndName : %s", err)
	}
	// fmt.Printf("%v.%v %v %v\n", SUBDOMAIN, DOMAIN, GODADDY_KEY, GODADDY_SECRET)
	return nil
}

func run() {
	ownIP, err := getOwnIPv4()
	if err != nil {
		// log.Fatal(err)
		fmt.Printf("run() ERR getOwnIPv4() %v\n", err)
	}
	domainIP, err := getDomainIPv4()
	if err != nil {
		// log.Fatal(err)
		fmt.Printf("run() ERR getDomainIPv4() %v\n", err)
	}
	fmt.Printf("%v -> %v\n", domainIP, ownIP)
	if domainIP != ownIP {
		if err := putNewIP(ownIP); err != nil {
			// log.Fatal(err)
			fmt.Printf("run() ERR putNewIP() %v\n", err)
		}
	}
}

// globals
var GODADDY_KEY = os.Getenv("GODADDY_KEY")
var GODADDY_SECRET = os.Getenv("GODADDY_SECRET")
var DOMAIN = os.Getenv("GODADDY_DOMAIN")
var SUBDOMAIN = os.Getenv("GODADDY_SUBDOMAIN")
var POLLING int64 = 360

func Dns(v ...string) {
	logFile := flag.String("log", "", "Path for log file (will be created if it doesn't exist)")
	if len(v) != 0 {
		fmt.Printf("v = %v\n", v)
		GODADDY_KEY = v[0]
		GODADDY_SECRET = v[1]
		DOMAIN = v[2]
		SUBDOMAIN = v[3]
	} else {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
		// required flags
		keyPtr := flag.String("key", os.Getenv("GODADDY_KEY"), "Godaddy API key")
		secretPtr := flag.String("secret", os.Getenv("GODADDY_SECRET"), "Godaddy API secret")
		pollingPtr := flag.Int64("interval", 360, "Polling interval in seconds. Lookup Godaddy's current rate limits before setting too low. Defaults to 360. (Optional)")
		domainPtr := flag.String("domain", os.Getenv("GODADDY_DOMAIN"), "Your top level domain (e.g., example.com) registered with Godaddy and on the same account as your API key")
		// optional flags
		subdomainPtr := flag.String("subdomain", os.Getenv("GODADDY_SUBDOMAIN"), "The data value (aka host) for the A record. It can be a 'subdomain' (e.g., 'subdomain' where 'subdomain.example.com' is the qualified domain name). Note that such an A record must be set up first in your Godaddy account beforehand. Defaults to @. (Optional)")
		flag.Parse()
		// fmt.Printf("%v %v\n", *domainPtr, domainPtr)
		SUBDOMAIN = *subdomainPtr
		DOMAIN = *domainPtr
		GODADDY_SECRET = *secretPtr
		GODADDY_KEY = *keyPtr
		POLLING = *pollingPtr
	}

	if *logFile == "" {
		log.SetOutput(os.Stdout)
	} else {
		f, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Couldn't open log file: %s", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	if DOMAIN == "" {
		log.Fatalf("You need to provide your domain")
	}

	if GODADDY_SECRET == "" {
		log.Fatalf("You need to provide your API secret")
	}

	if GODADDY_KEY == "" {
		log.Fatalf("You need to provide your API key")
	}

	// run
	for {
		run()
		fmt.Println("POLLING DNS v0.1.3")
		time.Sleep(time.Second * time.Duration(POLLING))
	}
}

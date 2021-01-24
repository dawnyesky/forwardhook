package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"encoding/json"
	"github.com/Jeffail/gabs"
)

const row_sep string = ";;;"

// maxRetries indicates the maximum amount of retries we will perform before
// giving up
var maxRetries = 10

// mirrorRequest will POST through body and headers from an
// incoming http.Request.
// Failures are retried up to 10 times.
func mirrorRequest(h http.Header, body []byte, url string) {
	attempt := 1
	for {
		fmt.Printf("Attempting %s try=%d\n", url, attempt)

		client := &http.Client{}

		// rB := bytes.NewReader(body)
		rB := bytes.NewBuffer(body)
		req, err := http.NewRequest("POST", url, rB)
		if err != nil {
			log.Println("[error] http.NewRequest:", err)
		}

		// Set headers from request
		req.Header = h

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Println("[error] client.Do:", err)
			time.Sleep(10 * time.Second)
		} else {
			resp.Body.Close()
			fmt.Printf("[success] %s status=%d\n", url, resp.StatusCode)
			break
		}

		attempt++
		if attempt > maxRetries {
			fmt.Println("[error] maxRetries reached")
			break
		}
	}
}

// parseSites gets sites out of the FORWARDHOOK_SITES environment variable.
// There is no validation at the moment but you can add 1 or more sites,
// separated by commas.
func parseSites() []string {
	sites := os.Getenv("FORWARDHOOK_SITES")

	if sites == "" {
		log.Fatal("No sites set up, provide FORWARDHOOK_SITES")
	}

	s := strings.Split(sites, ",")
	return s
}

func handleHook(sites []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rB, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Fail on ReadAll")
		}
		r.Body.Close()

		jsonParsed, err := gabs.ParseJSON(rB)
		if err != nil {
			log.Printf("Fail on gabs ParseJSON")
		}
		// Process data start
		items, ok := jsonParsed.Search("items").Data().([]interface{})
		if ok != true {
			log.Printf("Fail on gabs Search")
		}
		
		var titles, bodys, tags []string
		for _, item := range items {
			titles = append(titles, item.(map[string]interface {})["title"].(string))
			bodys = append(bodys, item.(map[string]interface {})["summary"].(map[string]interface {})["content"].(string))
			tags_str := item.(map[string]interface {})["categories"].([]interface {})
			var sub_tags []string
			for _, tag := range tags_str {
				tag_strs := strings.Split(tag.(string), "/")
				if tag_strs[len(tag_strs)-2] == "label" {
					sub_tags = append(sub_tags, tag_strs[len(tag_strs)-1])
				}
			}
			tags = append(tags, strings.Join(sub_tags, ","))
		}
		var data = map[string]interface {} {
			"value1":strings.Join(titles, row_sep),
			"value2":strings.Join(bodys, row_sep),
			"value3":strings.Join(tags, row_sep),
		}
		// Process data end
		new_rB, err := json.Marshal(data)
		if err != nil {
			log.Printf("Fail on Marshal")
		}

		for _, url := range sites {
			go mirrorRequest(r.Header, new_rB, url)
		}

		w.WriteHeader(http.StatusOK)
	})
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	sites := parseSites()
	fmt.Println("Will forward hooks to:", sites)

	http.Handle("/", handleHook(sites))
	http.HandleFunc("/health-check", handleHealthCheck)

	fmt.Printf("Listening on port 8000\n")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

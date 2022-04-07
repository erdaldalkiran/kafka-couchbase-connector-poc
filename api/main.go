package main

import (
	"encoding/json"
	"github.com/couchbase/gocb/v2"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"time"
)

var collection *gocb.Collection

func main() {

	initCouchbase()

	initHttpServer()
}

func initCouchbase() {
	host, set := os.LookupEnv("COUCHBASE_HOST")
	if !set {
		panic("COUCHBASE_HOST env is required")
	}

	user, set := os.LookupEnv("COUCHBASE_USERNAME")
	if !set {
		panic("COUCHBASE_USERNAME env is required")
	}

	pass, set := os.LookupEnv("COUCHBASE_PASSWORD")
	if !set {
		panic("COUCHBASE_PASSWORD env is required")
	}

	cluster, err := gocb.Connect(
		host,
		gocb.ClusterOptions{
			Username: user,
			Password: pass,
		})
	if err != nil {
		panic(err)
	}

	bucketName, set := os.LookupEnv("COUCHBASE_BUCKET")
	if !set {
		panic("COUCHBASE_BUCKET env is required")
	}

	collection = cluster.Bucket(bucketName).DefaultCollection()
}

func initHttpServer() {
	port, set := os.LookupEnv("API_PORT")
	if !set {
		panic("API_PORT env is required")
	}

	http.HandleFunc("/item", createItem)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}

func createItem(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		w.Header().Set("Content-Type", "application/json")

		id := uuid.NewString()
		data := map[string]interface{}{
			"id":             id,
			"name":           "ciko",
			"price":          13.75,
			"description":    "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			"active":         true,
			"occurrenceTime": time.Now().UTC(),
		}
		if _, err := collection.Insert(id, data, &gocb.InsertOptions{}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"err":`+err.Error()+`}`)
			return
		}

		w.WriteHeader(http.StatusCreated)
		body, _ := json.Marshal(data)
		w.Write(body)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

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

var cluster *gocb.Cluster
var itemCollection *gocb.Collection
var itemOutboxEventCollection *gocb.Collection

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

	var err error
	cluster, err = gocb.Connect(
		host,
		gocb.ClusterOptions{
			Username: user,
			Password: pass,
			TransactionsConfig: gocb.TransactionsConfig{
				DurabilityLevel: gocb.DurabilityLevelNone,
			},
		})
	if err != nil {
		panic(err)
	}

	bucketName, set := os.LookupEnv("COUCHBASE_BUCKET")
	if !set {
		panic("COUCHBASE_BUCKET env is required")
	}

	scopeName, set := os.LookupEnv("COUCHBASE_SCOPE")
	if !set {
		panic("COUCHBASE_SCOPE env is required")
	}

	collectionName, set := os.LookupEnv("COUCHBASE_COLLECTION")
	if !set {
		panic("COUCHBASE_COLLECTION env is required")
	}

	itemCollection = cluster.Bucket(bucketName).Scope(scopeName).Collection(collectionName)

	outboxCollectionName, set := os.LookupEnv("COUCHBASE_OUTBOX_COLLECTION")
	if !set {
		panic("COUCHBASE_OUTBOX_COLLECTION env is required")
	}
	itemOutboxEventCollection = cluster.Bucket(bucketName).Scope(scopeName).Collection(outboxCollectionName)
}

func initHttpServer() {
	port, set := os.LookupEnv("API_PORT")
	if !set {
		panic("API_PORT env is required")
	}

	http.HandleFunc("/create-item", createItem)
	http.HandleFunc("/update-item", updateItem)
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
			"version":        1,
			"name":           "ciko",
			"price":          13.75,
			"description":    "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			"active":         true,
			"occurrenceTime": time.Now().UTC(),
		}
		if _, err := itemCollection.Insert(id, data, &gocb.InsertOptions{}); err != nil {
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

func updateItem(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		id := req.URL.Query().Get("id")
		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{}

		_, err := cluster.Transactions().Run(func(ctx *gocb.TransactionAttemptContext) error {
			getResult, err := ctx.Get(itemCollection, id)
			if err != nil {
				return err
			}

			if err = getResult.Content(&response); err != nil {
				return err
			}

			version := response["version"]
			response["version"] = version.(float64) + 1

			_, err = ctx.Replace(getResult, response)
			if err != nil {
				return err
			}

			event := map[string]interface{}{
				"id":             response["id"],
				"version":        response["version"],
				"type":           "UPDATED",
				"occurrenceTime": time.Now().UTC(),
			}

			_, err = ctx.Insert(itemOutboxEventCollection, uuid.NewString(), event)
			if err != nil {
				return err
			}

			return nil
		}, nil)
		if err != nil {
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, `{"err":`+err.Error()+`}`)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(response)
		w.Write(body)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

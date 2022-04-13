#!/bin/bash

curl -s --location --request PUT 'http://localhost:8083/connectors/demo-couchbase-source/config' \
--header 'Content-Type: application/json' \
-d @./connector-config.json | jq

curl -s --location --request GET 'http://localhost:8083/connectors?expand=status' \
--header 'Accept: application/json' | jq


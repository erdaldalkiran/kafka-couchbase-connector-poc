#!/bin/bash

curl -s --location --request GET 'http://localhost:8083/connectors?expand=status' \
--header 'Accept: application/json' | jq


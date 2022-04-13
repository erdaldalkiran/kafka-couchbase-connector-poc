#!/bin/bash

id=$(curl -s --location --request POST 'http://localhost:8080/create-item' | jq -r .id)
echo "item created with id: $id"

echo "updating item: $id"
x=1
while [ $x -le 5 ]
do
  curl -s --location --request POST "http://localhost:8080/update-item?id=$id" | jq .
  echo ""
  x=$(( $x + 1 ))
done

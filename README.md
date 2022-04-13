# kafka-couchbase-connector-poc
Kafka Couchbase Connector POC

##requirements
docker, docker-compose, curl, jq
## how to run the demo
execute `docker-compose up -d --build` to provision environment  
docker compose will provision the couchbase server with an admin user and the required collections, but it will take same time.  
wait until `demo.item` and `demo.item_outbox_event` collections of the `demo` bucket are created in the couchbase server. 
you can visit [couchbase ui](http://localhost:8091/ui/index.html). user name is `Administrator` and password is `admin123!`  
go to `scripts` folder   
execute `./create-connector.sh` to create the couchbase connector on the kafka connect server  
execute `./check-connector.sh` to ensure all the tasks of the connector are in `RUNNING` status  
execute `./run-demo.sh` to create an item in the demo.item collection of the demo bucket 
and then update the created item 5 times 
and write the update events to into the demo.item_outbox_event collection of the demo bucket

you can see the published events in the [kafdrop](http://localhost:9000/topic/demo-topic/messages?partition=0&offset=0&count=100&keyFormat=DEFAULT&format=DEFAULT).

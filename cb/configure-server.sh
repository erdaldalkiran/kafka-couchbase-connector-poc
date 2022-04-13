set -m

/entrypoint.sh couchbase-server &

sleep 15

# Setup initial cluster/ Initialize Node
couchbase-cli cluster-init -c 127.0.0.1 --cluster-name $CLUSTER_NAME --cluster-username $COUCHBASE_ADMINISTRATOR_USERNAME \
  --cluster-password $COUCHBASE_ADMINISTRATOR_PASSWORD --services data,index,query,fts --cluster-ramsize 1024 --cluster-index-ramsize 1024 \
  --cluster-fts-ramsize 1024 --index-storage-setting default

sleep 15

# Setup Administrator username and password
curl -v http://127.0.0.1:8091/settings/web -d port=8091 -d username=$COUCHBASE_ADMINISTRATOR_USERNAME -d password=$COUCHBASE_ADMINISTRATOR_PASSWORD

sleep 15

# Setup Bucket
couchbase-cli bucket-create -c 127.0.0.1:8091 --username $COUCHBASE_ADMINISTRATOR_USERNAME \
  --password $COUCHBASE_ADMINISTRATOR_PASSWORD --bucket $COUCHBASE_BUCKET --bucket-type couchbase \
  --bucket-ramsize 256

sleep 15

# Setup Scope
couchbase-cli collection-manage -c 127.0.0.1:8091 --username $COUCHBASE_ADMINISTRATOR_USERNAME \
  --password $COUCHBASE_ADMINISTRATOR_PASSWORD --bucket $COUCHBASE_BUCKET \
  --create-scope demo

sleep 15

# Setup Collection
couchbase-cli collection-manage -c 127.0.0.1:8091 --username $COUCHBASE_ADMINISTRATOR_USERNAME \
  --password $COUCHBASE_ADMINISTRATOR_PASSWORD --bucket $COUCHBASE_BUCKET \
  --create-collection $COUCHBASE_SCOPE.$COUCHBASE_COLLECTION

sleep 15

# Setup Collection
couchbase-cli collection-manage -c 127.0.0.1:8091 --username $COUCHBASE_ADMINISTRATOR_USERNAME \
  --password $COUCHBASE_ADMINISTRATOR_PASSWORD --bucket $COUCHBASE_BUCKET \
  --create-collection $COUCHBASE_SCOPE.$COUCHBASE_OUTBOX_COLLECTION

sleep 15

fg 1

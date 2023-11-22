package driver

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client
var MongoDatabase *mongo.Database
var MongoCollection *mongo.Collection

func MustInitMongoClient(
	uri string,
	dbName string,
	collectionName string,
) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	// ping
	err = client.Ping(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	MongoClient = client
	MongoDatabase = client.Database(dbName)
	MongoCollection = client.Database(dbName).Collection(collectionName)
	// createIndex
	_, _ = MongoCollection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{
			"app_name": 1,
			"ip":       1,
		},
	})
	_, _ = MongoCollection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{
			"channel_id": 1,
			"app_name":   1,
		},
	})
	_, _ = MongoCollection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.M{
			"created_at": -1,
		},
	})
}

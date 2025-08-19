package store

import (
	"context"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoClient *mongo.Client
	MongoMsgs   *mongo.Collection
	RedisClient *redis.Client
	Ctx         = context.Background()
)

func InitStore(mongoURI string) error {

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return err
	}
	MongoClient = client
	MongoMsgs = client.Database("realtime-chat").Collection("messages")

	RedisClient = redis.NewClient(&redis.Options{
		Addr:        "localhost:6379",
		Password:    "",
		DB:          0,
	})
	_, err = RedisClient.Ping(Ctx).Result()
	if err != nil {
		return err
	}

	err = client.Connect(Ctx)
	if err != nil {
		return err
	}

	return nil
}

func CloseStore() {
	if MongoClient != nil {
		_ = MongoClient.Disconnect(Ctx)
	}
	if RedisClient != nil {
		_ = RedisClient.Close()
	}
}
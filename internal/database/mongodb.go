package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type User struct {
	Id       int64  `bson:"_id"`
	Name     string `bson:"name"`
	Nickname string `bson:"nickname"`
	Password string `bson:"password"`
}

type NextId struct {
	For string `bson:"_id"`
	Id  int64  `bson:"next_id"`
}
type NextIdCreate struct {
	For string `bson:"_id"`
	Id  int64  `bson:"next_id"`
}

func NewMongoClient() *mongo.Client {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatalln("MongoDB URI is not set in environment.")
	}

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalln("Failed to connect to MongoDB:", err)
	}
	return mongoClient
}

func AddUser(ctx context.Context, mongoClient *mongo.Client, redisClient *redis.Client, name, nickname, password string) error {
	userCollection := mongoClient.Database("mydb").Collection("users")
	nextIdCollection := mongoClient.Database("mydb").Collection("next_id")

	dbCtx, close := context.WithTimeout(ctx, 5*time.Second)
	defer close()

	var NextId NextId
	findErr := nextIdCollection.FindOne(dbCtx, bson.D{{Key: "_id", Value: "users"}}).Decode(&NextId)
	if findErr != nil {
		if dbCtx.Err() != nil {
			return dbCtx.Err()
		}

		// check data for initialization
		if findErr == mongo.ErrNoDocuments {
			nextIdCreate := NextIdCreate{
				For: "users",
				Id:  1,
			}
			if _, insertErr := nextIdCollection.InsertOne(dbCtx, nextIdCreate); insertErr != nil {
				return insertErr
			}
		}

		return findErr
	}

	passwordHash, HashErr := HashString(password)
	if HashErr != nil {
		return HashErr
	}

	user := User{
		Id:       NextId.Id,
		Name:     name,
		Nickname: nickname,
		Password: passwordHash,
	}

	_, insertErr := userCollection.InsertOne(dbCtx, user)
	if insertErr != nil {
		if dbCtx.Err() != nil {
			return dbCtx.Err()
		}
		return insertErr
	}

	_, updateErr := nextIdCollection.UpdateOne(dbCtx, bson.D{{Key: "_id", Value: "users"}}, bson.D{{Key: "$inc", Value: bson.D{{Key: "next_id", Value: int64(1)}}}})
	if updateErr != nil {
		if dbCtx.Err() != nil {
			return dbCtx.Err()
		}
		log.Fatal("Failed to update next_id:", updateErr)
	}

	AddUserToCache(ctx, redisClient, NextId.Id, user)
	AddUserToCache(ctx, redisClient, nickname, user)

	return nil
}

func FindUserById(ctx context.Context, mongoClient *mongo.Client, id int64) (*User, error) {
	dbCtx, close := context.WithTimeout(ctx, 5*time.Second)
	defer close()

	userCollection := mongoClient.Database("mydb").Collection("users")
	var foundUser User
	err := userCollection.FindOne(dbCtx, bson.D{{Key: "_id", Value: id}}).Decode(&foundUser)

	if err != nil {
		if dbCtx.Err() != nil {
			return &User{}, dbCtx.Err()
		}
		return &User{}, err
	}
	return &foundUser, nil
}

func FindUserByNickname(ctx context.Context, mongoClient *mongo.Client, nickname string) (*User, error) {
	dbCtx, close := context.WithTimeout(ctx, 5*time.Second)
	defer close()

	userCollection := mongoClient.Database("mydb").Collection("users")
	var foundUser User
	err := userCollection.FindOne(dbCtx, bson.D{{Key: "nickname", Value: nickname}}).Decode(&foundUser)

	if err != nil {
		if dbCtx.Err() != nil {
			return &User{}, dbCtx.Err()
		}
		return &User{}, err
	}
	return &foundUser, nil
}

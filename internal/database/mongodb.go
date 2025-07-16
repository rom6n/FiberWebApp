package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

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

func NewMongo() *mongo.Client {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatalln("MongoDB URI is not set in environment.")
	}

	db, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalln("Failed to connect to MongoDB:", err)
	}
	return db
}

func AddUser(db *mongo.Client, name, nickname, password string) error {
	userCollection := db.Database("mydb").Collection("users")
	nextIdCollection := db.Database("mydb").Collection("next_id")

	var NextId NextId
	findErr := nextIdCollection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: "users"}}).Decode(&NextId)
	if findErr != nil {
		return findErr
	}

	passwordHash, HashErr := HashString(password)
	if HashErr != nil {
		return HashErr
	}

	isRight, verifyErr := VerifyHash(password, passwordHash)
	if verifyErr != nil {
		fmt.Println("🔴 Password V-Error:", verifyErr)
	}
	fmt.Println("❔ is equals:", isRight)

	fmt.Println("🔢 Password hash:", passwordHash)

	_, insertErr := userCollection.InsertOne(context.TODO(), User{
		Id:       NextId.Id,
		Name:     name,
		Nickname: nickname,
		Password: passwordHash,
	})
	if insertErr != nil {
		return insertErr
	}

	_, updateErr := nextIdCollection.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: "users"}}, bson.D{{Key: "$inc", Value: bson.D{{Key: "next_id", Value: int64(1)}}}})
	if updateErr != nil {
		log.Fatal("Failed to update next_id:", updateErr)
	}

	return nil
}

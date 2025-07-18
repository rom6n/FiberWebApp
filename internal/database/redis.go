package database

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-json"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func NewRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return client
}

func AddUserToCache(ctx context.Context, redisClient *redis.Client, key any, value User) {
	dbContext, close := context.WithTimeout(ctx, 5*time.Second)
	defer close()

	// matching cache type
	switch key.(type) {
	case string:
	case int64:
	case any:
		fmt.Println("‼️Wrong key type for cache‼️")
		return
	}

	jsonValue, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		fmt.Printf("🔴 Error while marshaling user: %v 🔴\n", marshalErr)
		return
	}

	// adding user in cache
	status := redisClient.SetEx(dbContext, fmt.Sprintf("user:%v", key), jsonValue, 15*time.Minute)

	// looking for an error
	if msg, err := status.Result(); err != nil {
		fmt.Printf("🔴 Error add user in cache: %v ; %v 🔴\n", msg, err)
	}
}

func FindUserInCacheById(ctx context.Context, mongoClient *mongo.Client, redisClient *redis.Client, id int64) (*User, error) {
	dbCtx, close := context.WithTimeout(ctx, 5*time.Second)
	defer close()

	val, cacheErr := redisClient.Get(dbCtx, fmt.Sprintf("user:%v", id)).Result()

	if cacheErr != nil {
		if cacheErr == redis.Nil {
			user, mongoErr := FindUserById(ctx, mongoClient, id)
			if mongoErr != nil {
				return &User{}, mongoErr
			}

			AddUserToCache(ctx, redisClient, id, *user)
			return user, nil
		}
		return &User{}, cacheErr
	}

	var user User
	if jsonErr := json.Unmarshal([]byte(val), &user); jsonErr != nil {
		return &User{}, jsonErr
	}

	fmt.Println("user data from cache by id")
	return &user, nil
}

func FindUserInCacheByNickname(ctx context.Context, mongoClient *mongo.Client, redisClient *redis.Client, nickname string) (*User, error) {
	dbCtx, close := context.WithTimeout(ctx, 5*time.Second)
	defer close()

	val, cacheErr := redisClient.Get(dbCtx, fmt.Sprintf("user:%v", nickname)).Result()

	if cacheErr != nil {
		if cacheErr == redis.Nil {
			user, mongoErr := FindUserByNickname(ctx, mongoClient, nickname)
			if mongoErr != nil {
				return &User{}, mongoErr
			}

			AddUserToCache(ctx, redisClient, nickname, *user)
			return user, nil
		}
		return &User{}, cacheErr
	}

	var user User
	if jsonErr := json.Unmarshal([]byte(val), &user); jsonErr != nil {
		return &User{}, jsonErr
	}
	
	fmt.Println("user data from cache by nickname")
	return &user, nil
}

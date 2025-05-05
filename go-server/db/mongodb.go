package db

import (
	"context"
	"fmt"
	"log"
	"time"
	"websocket-go/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

var MongoDB *mongo.Database

var Collections struct {
	Users    *mongo.Collection
	Messages *mongo.Collection
}

func Connect() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	MongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
		return false
	}

	//check the connection
	if err := MongoClient.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB doens't answer:", err)
		return false
	}

	fmt.Println("MongoDB connected!")

	MongoDB = MongoClient.Database("chat_db")

	Collections.Users = MongoDB.Collection("users")
	Collections.Messages = MongoDB.Collection("messages")

	return true
}

func Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if MongoClient == nil {
		return nil
	}

	return MongoClient.Disconnect(ctx)
}

func LoginUser(username, password string) (bool, *models.User, error) {
	if Collections.Users == nil {
		Collections.Users = MongoDB.Collection("users")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := Collections.Users.FindOne(ctx, bson.M{"username": username, "password": password}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil //user doesnt exist
		}

		return false, nil, err
	}

	return true, &user, nil

}

func UserExists(username string) bool {

	//userCollection := MongoClient.Database("chat_db").Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"username": username}

	var result models.User

	err := Collections.Users.FindOne(ctx, filter).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false // user does not exist
		}
		log.Printf("Eroare la căutarea utilizatorului: %v", err)
		return false
	}

	return true
}

func RegisterUser(username, password, email string) (*models.User, error) {
	if UserExists(username) {
		log.Printf("User %s already exists!", username)
		return nil, fmt.Errorf("user %s already exists", username)
	}

	if Collections.Users == nil {
		log.Println("Users Collection isn't initialized!")
		return nil, fmt.Errorf("users Collection isn't initialized")
	}

	newUser := models.User{
		Username: username,
		Password: password,
		Email:    email,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := Collections.Users.InsertOne(ctx, newUser)
	if err != nil {
		log.Printf("Error adding new user: %v", err)
		return nil, fmt.Errorf("error adding new user: %v", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		newUser.ID = oid
	}

	return &newUser, nil
}

func GetUserID(username string) (primitive.ObjectID, error) {
	if Collections.Users == nil {
		Collections.Users = MongoDB.Collection("users")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"username": username}

	projection := bson.M{"_id": 1}
	opts := options.FindOne().SetProjection(projection)

	var result struct {
		ID primitive.ObjectID `bson:"_id"`
	}

	err := Collections.Users.FindOne(ctx, filter, opts).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return primitive.NilObjectID, fmt.Errorf("user %s doesn't exist", username)
		}
		return primitive.NilObjectID, fmt.Errorf("error searching user id: %v", err)
	}

	return result.ID, nil
}

func AddMessage(username, content string) (*models.Message, error) {
	if Collections.Messages == nil {
		Collections.Messages = MongoDB.Collection("messages")
	}

	now := time.Now()
	sender, err := GetUserID(username)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	newMessage := models.Message{
		Sender:    sender,
		Content:   content,
		Timestamp: now,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := Collections.Messages.InsertOne(ctx, newMessage)
	if err != nil {
		return nil, fmt.Errorf("error adding the message:%v", err)
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		newMessage.ID = oid
	}

	return &newMessage, nil
}

func GetMessages() ([]map[string]interface{}, error) {

	if Collections.Messages == nil {
		Collections.Messages = MongoDB.Collection("messages")
	}
	if Collections.Users == nil {
		Collections.Users = MongoDB.Collection("users")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$sort", Value: bson.M{"timestamp": 1}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "sender",
			"foreignField": "_id",
			"as":           "senderInfo",
		}}},
		bson.D{{Key: "$unwind", Value: "$senderInfo"}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":       1,
			"content":   1,
			"timestamp": 1,
			"username":  "$senderInfo.username",
		}}},
	}

	cursor, err := Collections.Messages.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("error aggregate messages: %v", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("error decoding results: %v", err)
	}

	return results, nil
}

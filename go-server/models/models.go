package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)


type User struct{
	ID		primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Username    string             `bson:"username" json:"username"`
	Password    string             `bson:"password" json:"password"` // Nu expune parola în JSON
	Email       string             `bson:"email" json:"email"`
}

type Message struct{
	ID 		primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Sender 	primitive.ObjectID  `bson:"sender" json:"sender"`
	Content   string             `bson:"content" json:"content"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
}
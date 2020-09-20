package main

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/oauth2"
	"log"
	"time"
)

// oAuthUserCollection points to mongodb collection for storing our data
var oAuthUserCollection *mongo.Collection

// mongoClient is mongodb client
var mongoClient *mongo.Client

// OAuthUser stored in mongodb
type OAuthUser struct {
	Email    string `json:"email"`
	UserInfo *oidc.UserInfo
	Token    *oauth2.Token
}

// setupMongo connects to the mongodb, creating database:users and collection:oauth
// with index for email
func setupMongo(mongoURI string) (err error) {

	mongoClient, err = mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("unable to make monodb client with %s due to err %v", mongoURI, err)
		return err
	}

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Fatalf("unable to connect monodb at %s due to err %v", mongoURI, err)
		return err
	}

	// create database and collection
	oAuthUserCollection = mongoClient.Database("users").Collection("oauth")

	// create index for email
	_, err = oAuthUserCollection.Indexes().CreateOne(context.Background(),
		mongo.IndexModel{
			Keys: bson.M{
				"email": 1,
			},
			Options: options.Index().SetUnique(true),
		},
	)
	if err != nil {
		log.Fatalf("Unable to create index email due to  %v", err)
		return err
	}
	return nil
}

// addUserToDB will update the user record in the db or update if already present
// record will be `upsert` to oauth package
func addUserToDB(user OAuthUser) error {

	upsert := true
	updateResult, err := oAuthUserCollection.UpdateOne(
		context.TODO(),
		bson.M{"email": user.Email},
		bson.M{"$set": user},
		&options.UpdateOptions{Upsert: &upsert},
	)
	if err != nil {
		return fmt.Errorf("unable to insert/update user %s in db due to %v", user.Email, err)
	}

	if updateResult.MatchedCount == 1 {
		log.Printf("UPDATED EXISTING USER: %s", user.Email)
	}
	if updateResult.UpsertedCount == 1 {
		log.Printf("INSERTED NEW USER: %s", user.Email)
	}

	return nil
}

// getUsersFromDB get users from the database and and put it in usersChannels for consumption
func getUsersFromDB(usersChannels chan OAuthUser) {
	defer close(usersChannels)

	findOptions := options.Find()
	cur, err := oAuthUserCollection.Find(context.TODO(), bson.D{{}}, findOptions)
	if err != nil {
		log.Printf("[ERROR]: Coun't get users from db due to %v\n", err)
		return
	}

	for cur.Next(context.TODO()) {
		var user OAuthUser
		err := cur.Decode(&user)
		if err != nil {
			log.Printf("[WARNING] unable to decode the output of user from mongodb due to %v", err)
			continue
		}
		usersChannels <- user
	}

	return

}

package main

import (
	"context"
	"fmt"
	oidc "github.com/coreos/go-oidc"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/fitness/v1"
	"google.golang.org/api/people/v1"
	"io/ioutil"
	"log"
	"net/http"
)

// config oauth2.Config for storing client credentials
var config *oauth2.Config

// oidcProvider used for the role purpose of getting email id
var oidcProvider *oidc.Provider

// scope we will be asking user for the following permissions
var scope = []string{
	fitness.FitnessActivityReadScope,
	fitness.FitnessBodyReadScope,
	oidc.ScopeOpenID,
	people.UserinfoProfileScope,
	people.UserinfoEmailScope,
	"profile",
	"email",
}

// setupOAuthClientCredentials will create oauth2.config and oidc.Provider
// using fileName the path to a json storing credentials obtained from google console
func setupOAuthClientCredentials(fileName string) (err error) {

	// read credentials obtained from google console and apt callback
	credentialsBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatalf("unable to read file %s : err %v", fileName, err)
		return err
	}

	// make oauth.config
	config, err = google.ConfigFromJSON(credentialsBytes, scope...)
	if err != nil {
		log.Fatalf("Unable to parse client credientials to oauth config: %v", err)
		return err
	}

	// make odic provider: used to get email id and stuff
	oidcProvider, err = oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		log.Fatalf("oidc: Unable to setup odic google rovider %v", err)
		return err
	}

	log.Println("Oauth2 Client Setup Completed")
	return err
}

// oAuthCallbackHandler handles the callback from google.
// Responsibilities: check state, get offline token, get oidc user info, save this information in db
func oAuthCallbackHandler(ctx *gin.Context) {

	// oauth state check
	session := sessions.Default(ctx)
	retrievedState := session.Get("state")
	if retrievedState != ctx.Query("state") {
		// check if session state is same as the state in teh url
		_ = ctx.AbortWithError(http.StatusUnauthorized, fmt.Errorf("oauth: state paramerter is not right %s", retrievedState))
		return
	}

	// get oauth TOKEN
	code := ctx.Query("code")
	token, err := config.Exchange(context.TODO(), code, oauth2.AccessTypeOffline)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("oauth code-token exchange failed due to %v", err))
		return
	}

	// make token source for using it in oidc call
	tokenSource := config.TokenSource(context.TODO(), token)
	userInfo, err := oidcProvider.UserInfo(context.Background(), tokenSource)
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("unable to fetch user info due to %v", err))
		return
	}

	// create data and insert in db
	user := OAuthUser{
		Email:    userInfo.Email,
		UserInfo: userInfo,
		Token:    token,
	}
	if addUserToDB(user) != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user": user.Email,
	})
}

// authoriseUserHandler redirects the user to appropriate url for auth
// sets "state" in the session cookie
func authoriseUserHandler(ctx *gin.Context) {

	// create random string for oauth state and store it in session
	session := sessions.Default(ctx)
	oauthState := getRandomString()
	session.Set("state", oauthState)
	err := session.Save()
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError,
			fmt.Errorf("unable to save state key in session %v", err))
		return
	}

	// redirect the user to consent page
	authorisationURL := config.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
	ctx.Redirect(http.StatusTemporaryRedirect, authorisationURL)
	return
}

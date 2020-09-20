package main

import (
	"fmt"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

func ErrorHandle() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		err := c.Errors.Last()
		if err == nil {
			return
		}

		c.JSON(c.Writer.Status(),
			gin.H{
				"error": fmt.Sprintf("Error at the backend %v", err),
			})
		return
	}
}

func main() {

	//setTimeZone
	err := setTimezone()
	if err != nil {
		return
	}

	// mongodb connection
	mongoURI := "mongodb://localhost:27017"
	err = setupMongo(mongoURI)
	if err != nil {
		return
	}

	// oauth connection
	err = setupOAuthClientCredentials("./credentials.json")
	if err != nil {
		return
	}

	router := gin.Default()

	// setup session cookie storage
	var store = sessions.NewCookieStore([]byte("secret"))
	router.Use(sessions.Sessions("goquestsession", store))

	// custom error handling TODO: add ui
	router.Use(ErrorHandle())

	// index page
	//router.Static("/css", "./static/css")
	//router.Static("/img", "./static/img")
	//router.LoadHTMLGlob("templates/*")

	router.GET("/list", list)
	router.GET("/login", authoriseUserHandler)
	router.GET("/auth", oAuthCallbackHandler)

	// Add the pprof routes
	//pprof.Register(router)

	_ = router.Run("127.0.0.1:9090")
}

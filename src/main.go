package main

import (
	"fmt"
	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
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

func health(ctx *gin.Context) {
	if mongoClient != nil && config != nil {
		ctx.JSON(http.StatusOK,
			gin.H{
				"health": "ok",
			})
	}
	ctx.JSON(http.StatusBadGateway,
		gin.H{
			"health": "lol",
		})
}

func main() {

	//setTimeZone
	err := setTimezone()
	if err != nil {
		return
	}

	// mongodb connection
	mongoURI := "mongodb://mongo:27017"
	err = setupMongo(mongoURI)
	if err != nil {
		return
	}

	// oauth connection
	err = setupOAuthClientCredentials("/etc/secret/credentials.json")
	if err != nil {
		return
	}

	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	// cache setup
	cacheStore := persistence.NewInMemoryStore(time.Minute)

	// setup session cookie storage
	var store = sessions.NewCookieStore([]byte("secret"))
	router.Use(sessions.Sessions("goquestsession", store))

	// custom error handling TODO: add ui
	router.Use(ErrorHandle())

	// static
	router.Static("/css", "./web/css")
	router.Static("/img", "./web/img")
	router.Static("/js", "./web/js")
	router.StaticFile("/favicon.ico", "./web/favicon.ico")
	router.LoadHTMLFiles("web/index.html")

	router.GET("/", cache.CachePage(cacheStore, 2*time.Minute, index)) // index page
	router.GET("/list/json", list)                                     // show information in json

	router.GET("/login", authoriseUserHandler) // to register
	router.GET("/auth", oAuthCallbackHandler)  // oauth callback

	router.GET("/health", health)
	// Add the pprof routes
	//pprof.Register(router)

	_ = router.Run("0.0.0.0:8080")
}

package main

import (
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
)

func list(ctx *gin.Context) {
	_ = sessions.Default(ctx)
	flushCache()

	result, err := getAll()
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	sort.Sort(allUserInfo(result))
	ctx.IndentedJSON(http.StatusOK, result)
}

func index(ctx *gin.Context) {
	_ = sessions.Default(ctx)
	result, err := getAll()
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	sort.Sort(allUserInfo(result))
	ctx.HTML(
		http.StatusOK,
		"index.html",
		result,
	)
}

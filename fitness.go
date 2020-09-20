package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/snabb/isoweek"
	"google.golang.org/api/fitness/v1"
	"google.golang.org/api/option"
	"net/http"
	"time"
)

type stepsDBElement struct {
	string
	int64
}

// getFitnessService returns the service object google fitness with apt permission to read steps
func getFitnessService(user OAuthUser) (*fitness.Service, error) {

	tokenSource := config.TokenSource(context.TODO(), user.Token)
	service, err := fitness.NewService(
		context.TODO(),
		option.WithScopes(fitness.FitnessActivityReadScope),
		option.WithTokenSource(tokenSource),
	)
	if err != nil {
		err = fmt.Errorf("fitness api error: unable create service \"%v\"", err)
	}
	return service, err
}

// getStepCount returns a single int representing the numbers of steps between startTime and endTime
func getStepCount(user OAuthUser, startTime time.Time, endTime time.Time) (int64, error) {
	service, err := getFitnessService(user)
	if err != nil {
		return -4, nil
	}

	// make a request to get steps
	stepsAggregateResult, err := service.Users.Dataset.Aggregate("me", &fitness.AggregateRequest{
		AggregateBy: []*fitness.AggregateBy{
			{
				DataSourceId: "derived:com.google.step_count.delta:com.google.android.gms:estimated_steps",
				DataTypeName: "com.google.step_count.delta",
			},
		},
		BucketByTime: &fitness.BucketByTime{
			DurationMillis: endTime.Sub(startTime).Milliseconds(),
		},
		EndTimeMillis:   endTime.UnixNano() / nanosPerMilli,
		StartTimeMillis: startTime.UnixNano() / nanosPerMilli,
	}).Do()
	if err != nil {
		err = fmt.Errorf("fitness api error: unable to fetch required data due to \"%v\"", err)
		return -3, err
	}

	var steps int64
	// extract the time of one bucket
	for _, bucket := range stepsAggregateResult.Bucket {
		for _, data := range bucket.Dataset {
			for _, point := range data.Point {
				for _, value := range point.Value {
					steps = value.IntVal
					goto endLoop
				}
			}
		}
	}

endLoop:
	return steps, err
}

// getStepCountCurrentWeek returns step count for the current week
func getStepCountCurrentWeek(user OAuthUser) (int64, error) {
	var currentYear, currentWeek = time.Now().In(timeLocation).ISOWeek()
	startOfWeek := isoweek.StartTime(currentYear, currentWeek, timeLocation)
	endOfWeek := startOfWeek.AddDate(0, 0, 7)
	return getStepCount(user, startOfWeek, endOfWeek)
}

// getStepCountCurrentWeek returns step count for the current day
func getStepCountCurrentDay(user OAuthUser) (int64, error) {
	year, month, day := time.Now().In(timeLocation).Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, timeLocation)
	endOfDay := startOfDay.AddDate(0, 0, 1)
	return getStepCount(user, startOfDay, endOfDay)
}

// geAllDetailsOfUser returns email-id, step count of the current week, step count of the current day
// It's supposed to use all the functionality we have
// TODO: update new functionality
func geAllDetailsOfUser(user OAuthUser) (string, int64, int64, error) {

	currentWeekCount, err := getStepCountCurrentWeek(user)
	if err != nil {
		return "", 0, 0, err
	}

	currentDayCount, err := getStepCountCurrentDay(user)
	if err != nil {
		return "", 0, 0, err
	}

	return user.Email, currentWeekCount, currentDayCount, err
}

// getStepCountWrapper puts the results of calling "getStepCountFunc" on "user" inside "resultQueue" channel
// TODO: wg.Done() didn't work, why?
// TODO: How to write this without using thread variable
func getStepCountWrapper(
	resultQueue chan stepsDBElement,
	user OAuthUser,
	getStepCountFunc func(authUser OAuthUser) (int64, error)) {

	steps, err := getStepCountFunc(user)
	if err != nil {
		resultQueue <- stepsDBElement{user.Email, -1}
		return
	}

	resultQueue <- stepsDBElement{user.Email, steps}
	return
}

func getAll() (map[string]int64, error) {

	// get all users in usersChannels
	usersChannels := make(chan OAuthUser)
	go getUsersFromDB(usersChannels)

	// resultQueue : the steps of each user will be stored in it
	resultQueue := make(chan stepsDBElement)
	numbersOfUsers := 0
	for user := range usersChannels {
		numbersOfUsers++
		go getStepCountWrapper(resultQueue, user, getStepCountCurrentWeek)
	}

	result := make(map[string]int64)
	for i := 0; i < numbersOfUsers; i++ {
		userSteps := <-resultQueue
		result[userSteps.string] = userSteps.int64
	}

	return result, nil
}
func list(ctx *gin.Context) {
	_ = sessions.Default(ctx)

	result, err := getAll()
	if err != nil {
		_ = ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ctx.IndentedJSON(http.StatusOK, result)
}

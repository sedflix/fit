package main

import (
	"context"
	"fmt"
	"github.com/snabb/isoweek"
	"google.golang.org/api/fitness/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
	"log"
	"strings"
	"time"
)

type userInfoElement struct {
	Name      string
	Email     string
	PhotoUrl  string
	StepsDay  int64
	StepsWeek int64
}

// byStepsWeek implements sort.Interface based on the userInfoElement.StepsWeek field.
type allUserInfo []userInfoElement

func (a allUserInfo) Len() int           { return len(a) }
func (a allUserInfo) Less(i, j int) bool { return a[i].StepsWeek > a[j].StepsWeek }
func (a allUserInfo) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// getFitnessService returns the service object google fitness with apt permission to read steps
func getFitnessService(user OAuthUser) (*fitness.Service, error) {

	tokenSource := getTokenSource(user)

	service, err := fitness.NewService(
		context.Background(),
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
func getAllStepsOfUser(user OAuthUser) (int64, int64, error) {

	currentWeekCount, err := getStepCountCurrentWeek(user)
	if err != nil {
		return 0, 0, err
	}

	currentDayCount, err := getStepCountCurrentDay(user)
	if err != nil {
		return 0, 0, err
	}

	return currentWeekCount, currentDayCount, err
}

func getProfilePicUrl(user OAuthUser) (Name string, Url string, err error) {
	tokenSource := getTokenSource(user)
	service, err := people.NewService(
		context.Background(),
		option.WithScopes(people.UserinfoProfileScope),
		option.WithTokenSource(tokenSource),
	)
	if err != nil {
		err = fmt.Errorf("people api error: unable create service \"%v\"", err)
		return Name, Url, err
	}

	person, err := service.People.Get("people/me").
		PersonFields("photos,names").Do()
	if err != nil {
		err = fmt.Errorf("people api error: unable get person info \"%v\"", err)
		return Name, Url, err
	}

	if len(person.Photos) > 0 {
		photo := person.Photos[len(person.Photos)-1]
		Url = photo.Url
	}
	if len(person.Names) > 0 {
		name := person.Names[len(person.Names)-1]
		Name = name.DisplayName
	}
	return Name, Url, err
}

func getDetails(user OAuthUser) (userInfoElement, error) {
	currentWeekCount, currentDayCount, err := getAllStepsOfUser(user)
	if err != nil {
		log.Printf("uable to get setap for %s due to %v", user.Email, err)
	}

	Name, profilePicUrl, err2 := getProfilePicUrl(user)

	if err2 != nil {
		Name = fmt.Sprintf("%v", err2)
		profilePicUrl = fmt.Sprintf("%v", err2)
	} else {
		err2 = err
	}
	profilePicUrl = strings.TrimSuffix(profilePicUrl, "=s100")

	return userInfoElement{
		Name:      Name,
		Email:     user.Email,
		PhotoUrl:  profilePicUrl,
		StepsDay:  currentDayCount,
		StepsWeek: currentWeekCount,
	}, err

}

// getDetailsWrapper puts the results of calling "getDetailsFunc" on "user" inside "resultQueue" channel
// TODO: wg.Done() didn't work, why?
// TODO: How to write this without using thread variable
func getDetailsWrapper(
	resultQueue chan userInfoElement,
	user OAuthUser,
	getDetailsFunc func(authUser OAuthUser) (userInfoElement, error)) {

	detail, err := getDetailsFunc(user)
	if err != nil {
		resultQueue <- detail
		return
	}

	resultQueue <- detail
	return
}

func getAll() ([]userInfoElement, error) {

	// get all users in usersChannels
	usersChannels := make(chan OAuthUser)
	go getUsersFromDB(usersChannels)

	// resultQueue : the steps of each user will be stored in it
	resultQueue := make(chan userInfoElement)
	numbersOfUsers := 0
	for user := range usersChannels {
		numbersOfUsers++
		go getDetailsWrapper(resultQueue, user, getDetails)
	}

	result := make([]userInfoElement, 0)
	for i := 0; i < numbersOfUsers; i++ {
		element := <-resultQueue
		result = append(result, element)
	}

	return result, nil
}

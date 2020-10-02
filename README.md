# fit
A step count leaderboard.

See the weekly steps count of all the registered people and motivate yourself to walk a bit more maybe.  

**How do we get your step count?**  
When you click on the "Register" button, we ask your permission to read your google fit data.
We make this leaderboard, by fetching your step counter in your registered google fit account.

**Why Google Fit?**
Different people have different activity trackers, but almost all of them can be synced with your google fit account.
So, we except you to install a side-app or activate google-fit sync with your activity tracker. I know, it's irritating.


## Installation

### Pre-requisite

- You should setup a client credentials on google console with appropriate redirect URI.
    - If your origin URL is `x`, your redirect URI will be `x/auth`
- Download the credentials json. 

### Local

- place credentials json in `/etc/secret/credentials.json`
```
# src folder
# run the mongo db using docker or any other method
docker run -d -p 27017:27017 --name mongo mongo 

# make sure that mongo resolves to at whatever ip mongodb is running
echo "0.0.0.0     mongo" >> /etc/hosts

# build the the code
go build

# run the code
./fit
```

### Compose

- place credentials json in `/etc/secret/credentials.json`
- run `docker-compose up`


### Helm

- place credentials json at `<x>`
- you are likely to change `ingress` in the values of the chart
```
# charts

# create secret do that our app can access the credentials
kubectl create secret generic google-creds --from-file=<x>

# add mongo db chart
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency build

# insall the chart
helm install fit .
```

## Acknowledgement

The UI of this app was inspired from the tutorials at https://tympanus.net/codrops/
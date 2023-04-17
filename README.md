# FileUpAPI Service  
A service for uploading a file to server via GRPC protocol. If the file is JSON, further server side processign is done.

**Note:** A GRPC client application for testing the server is included. 

**Note:** Server application is a CLI application and takes command line params to set port, host, dir (directory where uploaded files reside) and jsondir (directory where processed JSON files reside)

**Note** Test cases use GRPC client for initiating the upload to the server 

**Note:** Assumption is made that JSON processing is post processing and is not part of response to client.

**Note:** Assumption is made that JSON file has a known schema.

## Commentary
Clean architecture pattern is used. The application is divided in api layer, service layer and repository layer (repository layer in this scenario is only persisting files to hard disk). One way data flow is maintained with data flowing from api layer (only responsibility is payload validation), to service layer (only responsibility is to implement business logic), to repository layer (only responsibility is to interact with the database)

Memory footprint optimizations:

For uploading file, GRPC client streaming is used.

For JSON processing, token iterator based json unmarshaling is done on source JSON file.

## Build Instructions

### spinning up containers
docker-compose and Dockerfile are used to spin up the fileUpAPI GRPC API server container.
use:

`docker compose build` to build

`docker compose up` to run

`docker compose down` to remove containers

GRPC API is listening on http://0.0.0.0:8090

### building and running the cli application to test the GRPC server
build and run: `make client`

Above command will run a test client application that will upload a test file to GRPC server

Use `make dockershell` to log into the docker shell and then `cd tmp/server` to see the uploaded file and `cd/tmp/json` to see the processed JSON file (if the uploaded file is JSON). Here `/tmp/server` and `/tmp/json` are default locations. Custom paths can be set via `-dir` and `-jsondir` CLI flags. 

`make test` to run tests

No attempt is made to write extensive tests to get wider coverage. Only happy path tests implemented.

### JSON 
The test json file that is processed has the following schema:
```
[
    {
        "playerName":"Asterix",
        "avatarName": "johnny",
        "playerScore": 100,
        "lifeCount": 111,
        "game":"second life"
    },
    {
        "playerName":"Jane Doe",
        "avatarName": "elder-jane",
        "playerScore": 131,
        "lifeCount": 10,
        "game":"the last of us"
    },
    {
        "playerName":"Imhotep",
        "avatarName": "egyptian prince",
        "playerScore": 131,
        "lifeCount": 10,
        "game":"Assassins Creed IV"
    }

]
```


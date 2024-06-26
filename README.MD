# Google Drive Watcher and Cloud Function Deployment

This repository contains a Go-based Google Drive watcher and a Google Cloud Function that processes files uploaded to a specific Google Drive folder. The watcher monitors the folder and triggers the Cloud Function via Pub/Sub when a new file is uploaded.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Deployment](#deployment)

## Overview

The Google Drive Watcher monitors a specified Google Drive folder and sends a Pub/Sub notification when a new file is uploaded. The Cloud Function is triggered by these notifications and processes the files by moving them to a temporary folder, simulating processing, and then moving them to an output folder.

## Prerequisites

- Google Cloud SDK installed
- Docker installed
- Go installed (version 1.16 or later)
- A Google Cloud project with billing enabled
- Google Drive API enabled in your Google Cloud project
- Pub/Sub API enabled in your Google Cloud project

## Deployement 

### Cloud function:

```
gcloud functions deploy amplify-cloud-function \
  --runtime go120 \
  --trigger-topic amplify \
  --allow-unauthenticated \
  --entry-point=AmplifyFunction \
  --set-env-vars INPUT_FOLDER_ID=1FGmm27O_8D5DVr2qLGWrGgQEZGTEN5CD,TEMP_FOLDER_ID=1CcjlKfutvuTh7cG9HBcvVCVtO7i6XiAs,OUTPUT_FOLDER_ID=1Exp2CLZXg6uQMrMILvqYUrMIq4diZSA4,GOOGLE_APPLICATION_CREDENTIALS="./service-account-key.json",DRIVE_ID="your_drive_id" \
  --source=. \
  --project=algomate-400914


```

If the role for artifact is missing in GCP: 

```
gcloud projects add-iam-policy-binding algomate-400914\
  --member=serviceAccount:algomate-400914@appspot.gserviceaccount.com \
  --role=roles/artifactregistry.reader
```

### Cloud run (watcher)

Create a pub/sub topic for amplify if it does not exist: 

```
gcloud pubsub topics create amplify
gcloud pubsub topics create amplify-dead-letter-topic
gcloud pubsub subscriptions create amplify-dead-letter-subscription --topic=amplify-dead-letter-topic


```

Build & push the docker image: 

```
gcloud iam service-accounts keys create service-account-key.json \
    --iam-account amplify@algomate-400914.iam.gserviceaccount.com

docker build -t amplify-watcher \
  --build-arg DRIVE_FOLDER_ID=1FGmm27O_8D5DVr2qLGWrGgQEZGTEN5CD \
  --build-arg PUBSUB_TOPIC=amplify \
  --build-arg WEBHOOK_URL=https://us-central1-algomate-400914.cloudfunctions.net/amplify-cloud-function \
  --build-arg PROJECT_ID=algomate-400914 \
  .

docker tag amplify-watcher gcr.io/algomate-400914/amplify-watcher
docker push gcr.io/algomate-400914/amplify-watcher

gcloud run deploy amplify-watcher-service \
  --image gcr.io/algomate-400914/amplify-watcher \
  --region us-central1 \
  --platform managed \
  --allow-unauthenticated 







```
PROJECT_NUMBER=$(gcloud projects describe algomate-400914 --format="value(projectNumber)")
gcloud projects add-iam-policy-binding algomate-400914 \
  --member=serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com \
  --role=roles/run.admin

gcloud projects add-iam-policy-binding algomate-400914 \
  --member=serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com \
  --role=roles/viewer

gcloud projects add-iam-policy-binding algomate-400914 \
  --member=serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com \
  --role="roles/iam.serviceAccountUser"

If you face issues with auth run the following: 

```
gcloud auth configure-docker
```

```
gcloud projects add-iam-policy-binding your-project-id \
    --member='user:your-email@example.com' \
    --role='roles/storage.objectViewer'
```
Deploy: 

```
ls -la /root/
./watcher


```
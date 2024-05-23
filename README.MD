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

Run the cloud function:

```
gcloud functions deploy amplify-cloud-function \
  --runtime go116 \
  --trigger-http \
  --allow-unauthenticated \
  --set-env-vars TEMP_FOLDER_ID=temp,OUTPUT_FOLDER_ID=output,INPUT_FOLDER_ID=input \
  --project=amplify-424209

```

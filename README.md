# Facial Scan Microservices

## Overview

This repository is a work-in-progress project that implements a microservices architecture for a facial scan application. The project leverages Kubernetes for orchestration, Kafka for messaging, and Firebase for authentication and storage. Each microservice is designed to handle specific responsibilities within the overall application.

## Current Functionality

### API Gateway
- Handles incoming HTTP requests.
- Routes requests to appropriate services.
- Provides authentication middleware using Firebase.

### Auth Service
- Manages user authentication and registration.
- Interfaces with Firebase for user authentication.
- Provides endpoints for user registration and health checks.
- Listens to Kafka topics for user registration events and processes them.

### User Management Service
- Manages user profiles and accounts.
- Updates user profiles with additional information.
- Checks if a user has a username.
- Listens to Kafka topics for profile updates and username checks.

### Image Upload Service
- Handles image uploads and stores them in Google Cloud Storage.
- Produces messages to Kafka with the image URL for further processing.
- Listens to Kafka topics for image uploads and processes them.

## Planned Functionality

- **Image Processing Service**: This service will process uploaded images to calculate facial symmetry and other metrics. It will consume messages from Kafka, perform the necessary image analysis, and then produce a response message with the results.
- **Leaderboard Service**: This service will maintain a leaderboard of users based on their facial scan scores. It will consume messages from Kafka to update the leaderboard and provide endpoints to fetch leaderboard data.
- **Notification Service**: This service will handle sending notifications to users based on various events (e.g., when their image processing results are ready, when they achieve a new high score).

## Architecture

The application is structured as a collection of microservices, each deployed as a separate container in a Kubernetes cluster. Communication between services is handled via Kafka, ensuring decoupled and scalable interactions.

### Microservices

- **API Gateway**: Entry point for all client requests.
- **Auth Service**: Manages user authentication and registration.
- **User Management Service**: Handles user profile management.
- **Image Upload Service**: Manages image uploads and storage.

### Technologies Used

- **Kubernetes**: Orchestration of microservices.
- **Kafka**: Messaging between microservices.
- **Firebase**: Authentication and database storage.
- **Google Cloud Storage**: Storage for uploaded images.
- **Go (Golang)**: Programming language for the microservices.
- **Docker**: Containerization of microservices.

## Getting Started

### Prerequisites

- Docker
- Kubernetes (Minikube or a cloud provider)
- Kafka
- Firebase account with service account credentials

### Setup

1. **Clone the repository**:

    ```sh
    git clone https://github.com/your-username/facial-scan-microservices.git
    cd facial-scan-microservices
    ```

2. **Set up Kubernetes and Kafka**:
    - Ensure your Kubernetes cluster is running.
    - Deploy Kafka in your Kubernetes cluster or have access to a Kafka broker.

3. **Environment Variables**:
    - Create a `.env` file in each service directory with the necessary environment variables.

4. **Secrets**:
    - Store your Firebase service account credentials in the `secrets` directory of each service.

5. **Deploy the services**:

    ```sh
    kubectl apply -f path/to/deployment.yaml
    ```

6. **Run the services**:
    - Build and run each service using Docker.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any changes or improvements.

## License

This project is licensed under the MIT License.

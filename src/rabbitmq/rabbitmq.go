package rabbitmq

import (
	environment "github.com/nocodeleaks/quepasa/environment"
)

// init pre-warms the RabbitMQ client for the connection string configured via
// RABBITMQ_CONNECTIONSTRING, if present. This is optional — bots that configure
// their own connection string via the API will create their clients on demand.
// The pre-warm runs in a goroutine so it never blocks application startup.
func init() {
	connStr := environment.Settings.RabbitMQ.ConnectionString
	if connStr == "" {
		return
	}

	go GetRabbitMQClient(connStr)
// REMOVED: Automatic initialization in init() to prevent duplicate connections
// The RabbitMQ connection will be initialized lazily when GetRabbitMQClient() is called
// by each server instance through InitializeRabbitMQConnections()
// This avoids creating duplicate connections for the same connection string.

//func init() {
// Only set the default queue name from environment if provided
//	rabbitmq_queue := environment.Settings.RabbitMQ.Queue
//	if len(rabbitmq_queue) > 0 {
//		RabbitMQQueueDefault = rabbitmq_queue
//	}

	// Connection initialization is now handled by GetRabbitMQClient() when needed
	// No premature connection is created here
}

# Golang Syslog Receiver and Alerting

So I want to create a syslog receiver for my mikrotik router. So the mikrotik will send a log with specific prefix when there a DDOS attack. This golang app should receive the log and then forward the information into telegram bot notification. This also store the logs into a file for future analysis.

## Technical Implementation

You should use `https://github.com/mcuadros/go-syslog` as the library for handling the syslog server.

## Splitting worker

Do not block the syslog receive process. If there are event, just receive and throw it into worker and the worker will take the action (e.g Send telegram notification, stores to the file)

## Bot sending

You should throttle the bot sending (maybe 5 seconds) to avoid block access.

## Code Quality

Please write a comprehensive, good code quality, with well strucutred project. So it will easy maintained in the future. Also, make sure you use idiomatic golang for better best practice

## Env

For better and easier environment migration, please use .env file and store the configuration on it. And please load the configuration on specific config module so it will centralized and easy to maintain in the future.

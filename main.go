package main

import "timewise-notification/cron/jobs"

func main() {
	jobs.RegisterJobs()

	// Keep the program running
	select {}
}

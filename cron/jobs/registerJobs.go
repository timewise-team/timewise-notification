package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/timewise-team/timewise-models/models"
	"gopkg.in/gomail.v2"
	"io/ioutil"
	"net/http"
	"time"
)

// RegisterJobs registers all cron jobs
func RegisterJobs() {
	c := cron.New()

	// Add a sample job
	_, err := c.AddFunc("@every 5s", func() { sendNotification() })
	if err != nil {
		fmt.Println("Error adding cron job:", err)
		return
	}

	_, err = c.AddFunc("@every 10m", func() {
		clearExpiredLinkEmailRequests()
	})

	if err != nil {
		fmt.Println("Error adding cron job:", err)
		return
	}

	c.Start()
	fmt.Println("Cron jobs started")
}

func sendNotification() {
	fmt.Println("Starting cron job: sendNotification at", time.Now())

	unsentNotifications, err := GetUnsentNotifications()
	if err != nil {
		fmt.Println("Error getting unsent notifications:", err)
		return
	}

	if unsentNotifications == nil || len(unsentNotifications) == 0 {
		fmt.Println("No unsent notifications")
		return
	}

	for _, notification := range unsentNotifications {
		if notification.NotifiedAt != nil && notification.NotifiedAt.Before(time.Now()) {
			// Send notification
			fmt.Printf("Sending notification ID %d to email %s\n", notification.ID, notification.UserEmail.Email)

			// Send email
			err := SendEmail(notification.UserEmail.Email, "Notification", notification.Message)
			if err != nil {
				fmt.Println("Error sending email:", err)
				continue
			}

			// Update notification to sent
			err = updateNotificationToSent(notification.ID)
			if err != nil {
				fmt.Println("Error updating notification to sent:", err)
			}

			fmt.Printf("Notification %d sent successfully\n", notification.ID)
		}
	}
}

func GetUnsentNotifications() ([]models.TwNotifications, error) {
	resp, err := http.Get("https://dms.timewise.space/dbms/v1/notification")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var notifications []models.TwNotifications
	if err := json.Unmarshal(body, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

func updateNotificationToSent(notificationID int) error {
	url := fmt.Sprintf("https://dms.timewise.space/dbms/v1/notification/%d", notificationID)
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update notification: status code %d", resp.StatusCode)
	}

	return nil
}

func clearExpiredLinkEmailRequests() {
	fmt.Println("Starting cron job: clearExpiredLinkEmailRequests at", time.Now())

	err := DeleteLinkEmailRequest()
	if err != nil {
		fmt.Println("Error getting expired link email requests:", err)
		return
	}
}

func DeleteLinkEmailRequest() error {
	resp, err := http.Get("https://dms.timewise.space/dbms/v1/user_email/clear-expired")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func SendEmail(to string, subject string, body string) error {
	for i := 0; i < len(smtpConfigs); i++ {
		dialer := ConfigSMTP()
		if dialer == nil {
			return errors.New("failed to configure SMTP dialer")
		}
		m := gomail.NewMessage()
		m.SetHeader("From", dialer.Username)
		m.SetHeader("To", to)
		m.SetHeader("Subject", subject)
		m.SetBody("text/html", body)

		if err := dialer.DialAndSend(m); err != nil {
			fmt.Println("Error sending email with current SMTP", dialer.Username, "config:", err.Error(), "\n")
			currentSMTPIndex = (currentSMTPIndex + 1) % len(smtpConfigs)
			continue
		}

		return nil
	}

	return errors.New("failed to send email with all SMTP configurations")
}

func ConfigSMTP() *gomail.Dialer {
	config := smtpConfigs[currentSMTPIndex]
	return gomail.NewDialer(config.Host, config.Port, config.Email, config.Password)
}

var currentSMTPIndex = 0

var smtpConfigs = []struct {
	Host     string
	Port     int
	Email    string
	Password string
}{
	{"smtp.gmail.com", 587, "timewise.space@gmail.com", "dczt wlvd eisn cixf"},
	{"smtp.gmail.com", 587, "khanhhnhe170088@fpt.edu.vn", "cddn ujge aqlm xmjb"},
	{"smtp.gmail.com", 587, "khanhhn.hoang@gmail.com", "dgbx xyvw ciqg txbl"},
	{"smtp.gmail.com", 587, "khanhhnhe170088@fpt.edu.vn", "iaqw vmoj fxgb zzne"},
	{"smtp.gmail.com", 587, "thuandqhe170881@fpt.edu.vn", "whzq ivlb hevo jhdi"},
	{"smtp.gmail.com", 587, "builanviet@gmail.com", "lowo laid zgda chnc"},
	{"smtp.gmail.com", 587, "ngkkhanh006@gmail.com", "soet mdxg doio fmrt"},
}

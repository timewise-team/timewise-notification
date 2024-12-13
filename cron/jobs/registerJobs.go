package jobs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/timewise-team/timewise-models/dtos/core_dtos/schedule_participant_dtos"
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

	_, err = c.AddFunc("@every 1m", func() { clearExpiredLinkEmailRequests() })
	if err != nil {
		fmt.Println("Error adding cron job:", err)
		return
	}

	_, err = c.AddFunc("@every 10s", func() { checkReminder() })
	if err != nil {
		fmt.Println("Error adding cron job:", err)
		return
	}

	c.Start()
	fmt.Println("Cron jobs started")
}

func GetReminders() ([]models.TwReminder, error) {
	resp, err := http.Get("https://dms.timewise.space/dbms/v1/reminder")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var reminders []models.TwReminder
	if err := json.Unmarshal(body, &reminders); err != nil {
		return nil, err
	}

	return reminders, nil

}

func updateReminderToSent(reminderID int) error {
	url := fmt.Sprintf("https://dms.timewise.space/dbms/v1/reminder/%d/is_sent", reminderID)
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
		return fmt.Errorf("failed to update reminder: status code %d", resp.StatusCode)
	}

	return nil
}

func createMessage(reminder models.TwReminder) string {
	// Kiểm tra xem StartTime và EndTime có nil không trước khi format
	var startTime, endTime string
	if reminder.Schedule.StartTime != nil {
		startTime = reminder.Schedule.StartTime.Add(7 * time.Hour).Format("02/01/2006 15:04")
	} else {
		startTime = "N/A" // Hoặc có thể để là một giá trị mặc định khác
	}
	if reminder.Schedule.EndTime != nil {
		endTime = reminder.Schedule.EndTime.Add(7 * time.Hour).Format("02/01/2006 15:04")
	} else {
		endTime = "N/A" // Hoặc có thể để là một giá trị mặc định khác
	}

	return fmt.Sprintf(`
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Reminder Email</title>
        <style>
            body {
                font-family: Arial, sans-serif;
                background-color: #f4f4f9;
                margin: 0;
                padding: 0;
                color: #333;
            }

            .email-container {
                max-width: 600px;
                margin: 0 auto;
                padding: 20px;
                background-color: #ffffff;
                border-radius: 8px;
                box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
            }

            .email-header {
                font-size: 24px;
                font-weight: bold;
                color: #4CAF50;
                margin-bottom: 20px;
                text-align: center;
            }

            .email-body {
                font-size: 16px;
                line-height: 1.5;
                margin-bottom: 30px;
            }

            .email-footer {
                font-size: 14px;
                color: #777;
                text-align: center;
            }

            .message-text {
                color: #333;
                font-size: 16px;
                margin: 20px 0;
                padding: 10px;
                background-color: #f9f9f9;
                border-left: 5px solid #4CAF50;
            }

            .highlight {
                color: #4CAF50;
                font-weight: bold;
            }
        </style>
    </head>
    <body>
        <div class="email-container">
            <div class="email-header">
                Reminder Notification
            </div>
            <div class="email-body">
                <p>Hello <span class="highlight">%s</span>,</p>
                <p>This is a reminder for you:</p>
                <div class="message-text">
                    <p><strong>Workspace:</strong> %s</p>
                    <p><strong>Workspace Description:</strong> %s</p>
                    <p><strong>Schedule:</strong> %s</p>
                    <p><strong>Schedule Description:</strong> %s</p>
                    <p><strong>Start Time:</strong> %s</p>
                    <p><strong>End Time:</strong> %s</p>
                </div>
            </div>
            <div class="email-footer">
                <p>Thank you for using our service!</p>
            </div>
        </div>
    </body>
    </html>
    `, reminder.WorkspaceUser.UserEmail.Email,
		reminder.WorkspaceUser.Workspace.Title,
		reminder.WorkspaceUser.Workspace.Description,
		reminder.Schedule.Title,
		reminder.Schedule.Description,
		startTime,
		endTime)
}

func PushNotification(notifications models.TwNotifications) {
	url := "https://dms.timewise.space/dbms/v1/notification"
	jsonData, err := json.Marshal(notifications)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Create a new request with the JSON payload
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Failed to push notification:", resp.StatusCode)
	}

	fmt.Println("Notification pushed successfully")
}

func GetParticipantsByScheduleId(id int) ([]schedule_participant_dtos.ScheduleParticipantInfo, error) {
	resp, err := http.Get(fmt.Sprintf("https://dms.timewise.space/dbms/v1/schedule_participant/schedule/%d", id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var participants []schedule_participant_dtos.ScheduleParticipantInfo
	if err := json.Unmarshal(body, &participants); err != nil {
		return nil, err
	}

	return participants, nil

}

func checkReminder() {
	fmt.Println("Starting cron job: checkReminder at", time.Now())

	reminders, err := GetReminders()
	if err != nil {
		fmt.Println("Error getting reminders:", err)
		return
	}
	now := time.Now()

	nowFormatted := now.Format("2006-01-02 15:04:05")
	now, _ = time.Parse("2006-01-02 15:04:05", nowFormatted)
	fmt.Printf("Now: %s", now)
	// Lấy thời gian trước đó 2 phút không có múi giờ
	twoMinutesAgo := now.Add(-2 * time.Minute)
	fmt.Printf("Two minutes ago: %s", twoMinutesAgo)
	for _, reminder := range reminders {
		reminderTimeFormatted := reminder.ReminderTime.Format("2006-01-02 15:04:05")
		reminderTime, _ := time.Parse("2006-01-02 15:04:05", reminderTimeFormatted)
		fmt.Printf("Reminder time: %s, before now: %t, after two minutes ago: %t, is sent: %t\n", reminderTime, reminderTime.Before(now), reminderTime.After(twoMinutesAgo), reminder.IsSent)
		fmt.Println(reminder.ReminderTime)
		if reminderTime.After(twoMinutesAgo) && reminderTime.Before(now) && !reminder.IsSent {
			if reminder.Type == "only me" {
				// Send reminder
				fmt.Printf("Sending reminder ID %d to email %s", reminder.ID, reminder.WorkspaceUser.UserEmail.Email)
				// Update reminder to sent
				err := updateReminderToSent(reminder.ID)
				if err != nil {
					fmt.Println("Error updating reminder to sent:", err)
					continue
				}
				// Create message
				message := createMessage(reminder)
				// Send email
				err = SendEmail(reminder.WorkspaceUser.UserEmail.Email, "Reminder", message)
				if err != nil {
					fmt.Println("Error sending email:", err)
					continue
				}
				notificationMessage := fmt.Sprintf("Schedule %s is about to start at %s on %s", reminder.Schedule.Title, reminder.Schedule.StartTime.Format("15:04"), reminder.Schedule.StartTime.Format("02/01/2006"))
				notification := models.TwNotifications{
					UserEmailId:     reminder.WorkspaceUser.UserEmail.ID,
					Type:            "reminder",
					Message:         notificationMessage,
					IsRead:          false,
					RelatedItemId:   reminder.Schedule.ID,
					RelatedItemType: "schedule",
					ExtraData:       "",
					IsSent:          false,
					NotifiedAt:      &now,
				}
				PushNotification(notification)

			} else {
				// Send reminder
				fmt.Printf("Sending reminder ID %d to all participants of schedule ID %d", reminder.ID, reminder.Schedule.ID)
				// Update reminder to sent
				err := updateReminderToSent(reminder.ID)
				if err != nil {
					fmt.Println("Error updating reminder to sent:", err)
					continue
				}
				// Create message
				message := createMessage(reminder)
				participants, err := GetParticipantsByScheduleId(reminder.Schedule.ID)
				if err != nil {
					fmt.Println("Error getting participants:", err)
					continue
				}
				fmt.Println("Participants:", participants)
				// Send email
				for _, participant := range participants {
					err = SendEmail(participant.Email, "Reminder", message)
					if err != nil {
						fmt.Println("Error sending email:", err)
						continue
					}
					notificationMessage := fmt.Sprintf("Schedule %s is about to start at %s on %s", reminder.Schedule.Title, reminder.Schedule.StartTime.Format("15:04"), reminder.Schedule.StartTime.Format("02/01/2006"))
					notification := models.TwNotifications{
						UserEmailId:     participant.UserId,
						Type:            "reminder",
						Message:         notificationMessage,
						IsRead:          false,
						RelatedItemId:   reminder.Schedule.ID,
						RelatedItemType: "schedule",
						ExtraData:       "",
						IsSent:          false,
						NotifiedAt:      &now,
					}
					PushNotification(notification)
				}
			}

		}
	}
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
	{"smtp.gmail.com", 587, "khanhhn@metric.vn", "pjax xcvp ohmf prgx"},
}

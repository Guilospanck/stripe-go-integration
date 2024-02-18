package repository

import (
	"fmt"

	"github.com/stripe/stripe-go/v76"
)

type User struct {
	Name                string `json:"name"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	SubscriptionStatus  string `json:"subscriptionStatus"`
	ExpireDateTimestamp int64  `json:"expireDateTimestamp"`
}

func UpdateUserAccount(user User) {
	fmt.Println()
	fmt.Printf("Updated user %+v account!", user)
	fmt.Println()
}

func GetUserFromDB(email string) (User, error) {
	user := User{
		Email:               email,
		Name:                "User from DB",
		ExpireDateTimestamp: 1707820250079,
		Password:            "",
		SubscriptionStatus:  "Active",
	}

	return user, nil
}

func CustomerAlreadyInTheDB(customerEmail string) (*User, bool) {
	// getUserFromDB(customerEmail)

	// checks if the customer is already an user
	return nil, false
}

func CreateUserAccount(name string, email string, subscriptionStatus stripe.SubscriptionStatus, expireDateTimestamp int64) User {
	password := _generateUserTemporaryPassword()

	user := User{Email: email, Name: name, Password: password, SubscriptionStatus: string(subscriptionStatus), ExpireDateTimestamp: expireDateTimestamp}

	// save customer data to database
	fmt.Println()
	fmt.Printf("User %+v account created!", user)
	fmt.Println()

	return user
}

func SendUserEmail(email, password string) {
	// send email to user with his temporary credentials
	fmt.Println()
	fmt.Printf("Email sent to %s with his new credentials: %s!", email, password)
	fmt.Println()
}

func _generateUserTemporaryPassword() string {
	// generates temporary password
	password := "apple-potato-mirror"

	fmt.Println()
	fmt.Printf("Generated %s\n", password)
	fmt.Println()

	return password
}

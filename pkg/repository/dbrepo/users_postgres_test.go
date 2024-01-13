//go:build integeration

package dbrepo

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
	"webapp/pkg/data"
	"webapp/pkg/repository"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	host     = "localhost"
	user     = "postgres"
	password = "postgres"
	dbName   = "users_test"
	port     = "5435"
	dsn      = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5"
)

var resource *dockertest.Resource
var pool *dockertest.Pool
var testDB *sql.DB
var testRepo repository.DatabaseRepo

func TestMain(m *testing.M) {
	// connect to docker if docker not running fail
	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not connect to docker; %s", err)
	}
	pool = p

	// setup docker options, specifying the image and so forth
	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14.5",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}
	// get a resource (docker image)
	resource, err := pool.RunWithOptions(&opts)
	if err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not start resource: %s", err)
	}
	// start the image and wait until it's ready
	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbName))
		if err != nil {
			log.Println("Error: ", err)

			return err
		}

		return testDB.Ping()
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect to database: %s", err)
	}

	// populate database with empty tables
	err = createTables()
	if err != nil {
		log.Fatalf("error creating tables: %s", err)
	}

	//
	testRepo = &PostgresDBRepo{
		DB: testDB,
	}

	// run the tests
	code := m.Run()

	// clean up
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	os.Exit(code)
}

func createTables() error {
	tableSQL, err := os.ReadFile("./testdata/users.sql")

	if err != nil {
		fmt.Println(err)
		return err
	}

	_, err = testDB.Exec(string(tableSQL))
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func Test_pingDB(t *testing.T) {
	err := testDB.Ping()

	if err != nil {
		t.Error("can't ping database")
	}
}

func TestPostgresDBRepo_InsertUser(t *testing.T) {
	testUser := data.User{
		FirstName: "Mohammad",
		LastName:  "Mohammadi",
		Email:     "Mohammad@gmail.com",
		Password:  "test",
		IsAdmin:   1,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	id, err := testRepo.InsertUser(testUser)

	if err != nil {
		t.Errorf("insert user returned an error: %s", err)
	}

	if id != 1 {
		t.Errorf("insert user returned wrong id expected 1; got %d", id)
	}
}

func TestPostgresDBRepo_AllUsers(t *testing.T) {
	users, err := testRepo.AllUsers()

	if err != nil {
		t.Errorf("all users returned an error: %s", err)
	}

	if len(users) != 1 {
		t.Errorf("all users returned wrong size expected 1; got %d", len(users))
	}

	testUser := data.User{
		FirstName: "Hey",
		LastName:  "Hey",
		Email:     "Hey@gmail.com",
		Password:  "test",
		IsAdmin:   1,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	_, _ = testRepo.InsertUser(testUser)
	users, err = testRepo.AllUsers()

	if err != nil {
		t.Errorf("all users returned an error: %s", err)
	}

	if len(users) != 2 {
		t.Errorf("all users returned wrong size expected 2; got %d", len(users))
	}
}

func TestPostgresDBRepo_GetUser(t *testing.T) {
	user, err := testRepo.GetUser(1)
	if err != nil {
		t.Errorf("error getting user by id: %s", err)
	}

	if user.Email != "Mohammad@gmail.com" {
		t.Errorf("wrong email returned; expected Mohammad@gmail.com but got %s", user.Email)
	}

	user, err = testRepo.GetUser(3)
	if err == nil {
		t.Error("no error reported when getting none-existing user by id")
	}
}

func TestPostgresDBRepo_GetUserByEmail(t *testing.T) {
	user, err := testRepo.GetUserByEmail("Hey@gmail.com")
	if err != nil {
		t.Errorf("error getting user by id: %s", err)
	}

	if user.Email != "Hey@gmail.com" {
		t.Errorf("wrong email returned; expected 2 but got %d", user.ID)
	}

	user, err = testRepo.GetUserByEmail("notexists@gmail.com")
	if err == nil {
		t.Error("no error reported when getting none-existing user by email")
	}
}

func TestPostgresDBRepo_UpdateUser(t *testing.T) {
	user, _ := testRepo.GetUser(2)

	user.FirstName = "Jane"
	user.Email = "Jane@gmail.com"

	err := testRepo.UpdateUser(*user)
	if err != nil {
		t.Errorf("error updating user %d: %s", 2, err)
	}

	user, _ = testRepo.GetUser(2)
	if user.FirstName != "Jane" || user.Email != "Jane@gmail.com" {
		t.Errorf("expected updated record to have first_name Jane and Email Jane@gmail.com but got %s and %s", user.FirstName, user.Email)
	}
}

func TestPostgresDBRepo_DeleteUser(t *testing.T) {
	err := testRepo.DeleteUser(2)
	if err != nil {
		t.Errorf("error deleting user %d: %s", 2, err)
	}

	_, err = testRepo.GetUser(2)
	if err == nil {
		t.Error("retrieved user id 2 who should have been deleted")
	}
}

func TestPostgresDBRepo_ResetPassword(t *testing.T) {
	err := testRepo.ResetPassword(1, "newpass")
	if err != nil {
		t.Error("error resetting user password")
	}

	user, _ := testRepo.GetUser(1)
	matches, err := user.PasswordMatches("newpass")
	if err != nil {
		t.Error(err)
	}

	if !matches {
		t.Errorf("password should match `newpass` but does not")
	}
}

func TestPostgresDBRepo_InsertUserImage(t *testing.T) {
	var image data.UserImage

	image.UserID = 1
	image.FileName = "test.png"
	image.CreatedAt = time.Now()
	image.UpdatedAt = time.Now()

	newID, err := testRepo.InsertUserImage(image)
	if err != nil {
		t.Error("inserting user image failed: ", err)
	}

	if newID != 1 {
		t.Error("got wrong id for image; should be one but got ", newID)
	}

	image.UserID = 100
	_, err = testRepo.InsertUserImage(image)
	if err == nil {
		t.Error("inserted a user image with none-existing user_id")
	}
}

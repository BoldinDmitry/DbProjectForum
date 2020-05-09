package configs

var PostgresPreferences postgresPreferencesStruct

func init() {
	PostgresPreferences = postgresPreferencesStruct{
		User:     "postgres",
		Password: "",
		Port:     "5432",
	}
}

package configs

var PostgresPreferences postgresPreferencesStruct

func init() {
	PostgresPreferences = postgresPreferencesStruct{
		User:     "postgres",
		Password: "",
		DBName:   "postgres",
		Port:     "5432",
	}
}

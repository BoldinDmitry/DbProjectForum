package configs

var PostgresPreferences postgresPreferencesStruct

func init() {
	PostgresPreferences = postgresPreferencesStruct{
		User:     "docker",
		Password: "docker",
		DBName:   "docker",
		Port:     "5432",
	}
}

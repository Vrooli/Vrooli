package envx

import "os"

func lookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

package main

func FatalErr(err error) {
	if err != nil {
		Log.Fatalln(err.Error())
	}
}

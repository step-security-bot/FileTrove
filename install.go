package filetrove

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// InstallFT creates and downloads necessary directories and databases and copies them to installPath
func InstallFT(installPath string, version string, initdate string) (error, error, error, error, error) {
	var choice string

	// Printing an additional newline
	fmt.Println()

	fmt.Println("Creating database and logfile directories.")
	dbdirerr := os.Mkdir(filepath.Join(installPath, "db"), os.ModePerm)
	if dbdirerr != nil {
		return dbdirerr, nil, nil, nil, nil
	}
	logsdirerr := os.Mkdir(filepath.Join(installPath, "logs"), os.ModePerm)
	if logsdirerr != nil {
		return nil, logsdirerr, nil, nil, nil
	}
	fmt.Println("Creating filetrove database.")
	trovedberr := CreateFileTroveDB(filepath.Join(installPath, "db"), version, initdate)
	if trovedberr != nil {
		return nil, nil, trovedberr, nil, nil
	}
	fmt.Println("Downloading signature database.")
	siegfriederr := GetSiegfriedDB(installPath)

	fmt.Print("\nNext step is to download the NSRL database which is 1.4 GB compressed. Proceed? [y/n]: ")
	_, err := fmt.Scan(&choice)
	if err != nil {
		os.Exit(-1)
	}

	choice = strings.TrimSpace(choice)
	choice = strings.ToLower(choice)

	var nsrlerr error
	if choice == "y" {
		nsrlerr = GetNSRL(installPath)
		zippedFile := filepath.Join(installPath, "db", "nsrl.db.gz")
		fmt.Println("\nUnzipping NSRL database.")
		nsrlerr = UnzipNSRL(zippedFile, filepath.Join(installPath, "db"))
		if nsrlerr == nil {
			println()
			fmt.Println("NSRL database extracted. You can safely delete nsrl.db.gz in the db directory.")
		}
	}

	if choice == "n" {
		log.Println("Skipping NSRL download. You have to copy an existing nsrl.db into the db directory.")
	}

	return dbdirerr, logsdirerr, trovedberr, siegfriederr, nsrlerr
}

// CheckInstall checks if all necessary file are available
func CheckInstall(version string) error {
	_, err := os.Stat(filepath.Join("db", "siegfried.sig"))
	if os.IsNotExist(err) {
		fmt.Println("ERROR: siegfried signature file not installed.")
	}
	_, err = os.Stat(filepath.Join("db", "filetrove.db"))
	if os.IsNotExist(err) {
		fmt.Println("ERROR: filetrove database does not exist.")
	}
	_, dberr := os.Stat(filepath.Join("db", "nsrl.db"))
	if os.IsNotExist(dberr) {
		fmt.Println("ERROR: nsrl database does not exist.")
	}

	if dberr == nil {
		ftdb, connerr := ConnectFileTroveDB("db")
		if connerr != nil {
			fmt.Println("Could not connect or open database. Error: " + connerr.Error())
			os.Exit(1)
		}

		compatible, dbversion, checkerr := CheckVersion(ftdb, version)
		if checkerr != nil {
			fmt.Println("Could not check database version. Error: " + checkerr.Error())
		}
		if !compatible {
			fmt.Println("Database not compatible with this Version of FileTrove. Database version: " + dbversion)
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Println("ERROR: Some or more checks failed, FileTrove is not ready. Did you run the installation?")
		return err
	}

	return nil
}

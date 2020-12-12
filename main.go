package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func handleError(err error) {
	if err != nil {
		pterm.DefaultSpinner.Fail(err)
		os.Exit(1)
	}
}

func main() {

	pterm.DefaultSection.Println("Этот скрипт установит на ваш компьютер \n  приложение shutter и модуль для загрузки скриншотов на w6p.ru")

	pterm.DefaultSection.WithLevel(2).Println("Для продолжения установки введите root пароль")

	sudo := exec.Command("/bin/bash", "-c", "sudo su")
	sudoOut, err := sudo.CombinedOutput()
	fmt.Printf("%s\n", sudoOut)

	if err != nil {
		log.Fatal()
	}

	// Добавление репозитория
	repoAdd()

	// Установка зависимостей
	depsInstall()

	// Установка shutter
	shutterInstall()

	// установка модуля w6p
	installModules()

	// Обновление бинарника
	fixShutter()

	pterm.DefaultSection.WithLevel(2).Println("Все готово! \n   Для удобства использования, вы можете установить в настройках системы действие на сочетание Ctrl+PrtSc, \n   указав в качестве команды: shutter -s")
}

func repoAdd() {
	spinnerRepoAdd, _ := pterm.DefaultSpinner.Start("Добавление репозитория")
	repo := exec.Command("/bin/bash", "-c", "sudo add-apt-repository ppa:linuxuprising/shutter -y")
	_, err := repo.CombinedOutput()

	if err != nil {
		spinnerRepoAdd.UpdateText("Не удалось добавить репозиторий")
		spinnerRepoAdd.Fail()
		os.Exit(1)
	}
	spinnerRepoAdd.Success()
}

func depsInstall() {
	pterm.Info.Println("Установка зависимостей")
	spinnerDep, _ := pterm.DefaultSpinner.Start()

	spinnerDep.UpdateText("Обновление кеша")
	aptUpdate := exec.Command("/bin/bash", "-c", "sudo apt update")
	_, err := aptUpdate.CombinedOutput()
	if err != nil {
		spinnerDep.UpdateText("Не удалось обновить кеш")
		spinnerDep.Warning()
	}

	deps := countDeps()

	for i := 0; i < len(deps); i++ {
		dependence := strings.TrimSpace(deps[i])
		spinnerDep.UpdateText("Установка пакета " + dependence)

		installDeps := exec.Command("/bin/bash", "-c", "sudo apt install -y "+dependence)
		_, err := installDeps.CombinedOutput()
		if err != nil {
			spinnerDep.UpdateText("Не удалось установить пакет" + dependence)
			spinnerDep.Warning()
		}
	}

	spinnerDep.Success("Зависимости установлены")
}

func shutterInstall() {
	spinnerInstall, _ := pterm.DefaultSpinner.Start("Установка shutter")
	install := exec.Command("/bin/bash", "-c", "sudo apt install shutter -y")
	_, _ = install.CombinedOutput()

	spinnerInstall.Success()
}

func countDeps() []string {
	cmd := exec.Command("/bin/bash", "-c", "apt-cache depends shutter | grep Зависит | sed 's/Зависит: //g'")

	stdout, err := cmd.StdoutPipe()
	handleError(err)

	err = cmd.Start()
	handleError(err)

	buff := bufio.NewScanner(stdout)
	var count []string

	for buff.Scan() {
		count = append(count, buff.Text()+"\n")
	}

	if len(count) == 0 {
		handleError(errors.New("Не удалось получить зависимости"))
		os.Exit(1)
	}

	return count
}

func installModules() {
	moduleDir := "/usr/share/shutter/resources/system/upload_plugins/upload"

	deleteOldModules(moduleDir)

	w6pAdd, _ := pterm.DefaultSpinner.Start("Установка модуля w6p")
	load := download("https://raw.githubusercontent.com/varrcan/shutter-w6p/master/W6p.pm")
	handleError(load)

	mv := exec.Command("/bin/bash", "-c", "sudo mv W6p.pm "+moduleDir)

	_, err := mv.CombinedOutput()
	handleError(err)

	time.Sleep(time.Second * 1)
	w6pAdd.Success()
}

func fixShutter() {
	w6pAdd, _ := pterm.DefaultSpinner.Start("Обновление бинарного файла")
	load := download("https://raw.githubusercontent.com/varrcan/shutter-w6p/master/shutter")
	handleError(load)

	mv := exec.Command("/bin/bash", "-c", "sudo mv shutter /usr/bin")

	_, err := mv.CombinedOutput()
	handleError(err)

	time.Sleep(time.Second * 1)
	w6pAdd.Success()
}

func baseSettings() {
	settings, _ := pterm.DefaultSpinner.Start("Установка стандартных настроек")
	load := download("https://raw.githubusercontent.com/varrcan/shutter-w6p/master/.shutter.zip")
	handleError(load)

	_, err := Unzip(".shutter.zip", "")
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second * 1)
	settings.Success()
}

func deleteOldModules(directory string) {
	deleteModules, _ := pterm.DefaultSpinner.Start("Удаление стандартных модулей")
	if _, err := os.Stat(directory); err != nil {
		if os.IsNotExist(err) {
			handleError(err)
		}
	}

	exec := exec.Command("/bin/bash", "-c", "sudo rm -rf "+directory+"/*")

	_, err := exec.CombinedOutput()
	handleError(err)

	deleteModules.Success()
}

func download(url string) (err error) {
	filename := path.Base(url)

	resp, httpErr := http.Get(url)
	handleError(httpErr)

	defer resp.Body.Close()

	file, fileErr := os.Create(filename)
	handleError(fileErr)

	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return
}

func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

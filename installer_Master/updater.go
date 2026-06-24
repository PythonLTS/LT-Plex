package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	zipPath := "update.zip"
	targetDir := "/opt/ltplex/app"

	err := unzipAppFolder(zipPath, targetDir)
	if err != nil {
		fmt.Printf("Ошибка при обновлении: %v\n", err)
		os.Exit(1)
	}

	// Тот самый заветный done
	fmt.Print("done")
}

func unzipAppFolder(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		// Проверяем, что файл находится внутри папки app/
		if !strings.HasPrefix(file.Name, "LT-Plex-main/app/") {
			continue
		}

		// Убираем префикс "app/", чтобы распаковывать содержимое прямо в /opt/ltplex/app
		// Если тебе нужно, чтобы внутри /opt/ltplex/app создавалась еще одна папка app/, просто удали эту строку
		relPath := strings.TrimPrefix(file.Name, "LT-Plex-main/app/")
		if relPath == "" {
			continue
		}

		path := filepath.Join(target, relPath)

		// Если это директория, создаем её
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
			continue
		}

		// На случай, если папка для файла еще не создана
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Создаем целевой файл
		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		// Открываем файл из архива
		fileInZip, err := file.Open()
		if err != nil {
			targetFile.Close()
			return err
		}

		// Копируем данные
		_, err = io.Copy(targetFile, fileInZip)
		targetFile.Close()
		fileInZip.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
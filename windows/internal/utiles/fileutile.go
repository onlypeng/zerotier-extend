package utiles

import (
	"fmt"
	"io"
	"os"
)

// 获取文件中文本内容
func GetFileContent(filePath string) (string, error) {
	return "", nil
}

func CopyFile(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制文件内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 确保文件内容写入磁盘
	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("同步文件内容失败: %w", err)
	}

	return nil
}

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	// 定义命令行参数
	inputPattern := flag.String("input", "./*", "输入路径（支持通配符，如 demo.avi, C:/video/*, C:/video/**/*）")
	outputDir := flag.String("output", "./output", "输出目录（保存处理后的视频文件）")
	watermarkPath := flag.String("watermark", "", "水印图片路径（可选）")
	watermarkWidth := flag.Int("width", 100, "水印宽度（像素）")
	watermarkHeight := flag.Int("height", -1, "水印高度（像素），-1 表示按比例调整")
	watermarkX := flag.String("x", "W-w-10", "水印水平位置（像素或表达式，如 10 或 W-w-10）")
	watermarkY := flag.String("y", "10", "水印垂直位置（像素或表达式，如 10 或 H-h-10）")
	maxConcurrency := flag.Int("concurrency", 4, "最大并发数")
	crfThreshold := flag.Int64("crf-threshold", 10*1024*1024, "文件大小阈值（字节），大于此值时添加 -crf 28")

	// 解析命令行参数
	flag.Parse()

	// 确保输出目录存在
	if err := os.MkdirAll(*outputDir, os.ModePerm); err != nil {
		log.Fatalf("无法创建输出目录: %v", err)
	}

	// 获取匹配的文件列表
	matches, err := filepath.Glob(*inputPattern)
	if err != nil {
		log.Fatalf("无法解析输入路径模式: %v", err)
	}
	if len(matches) == 0 {
		log.Fatalf("未找到匹配的文件: %s", *inputPattern)
	}

	// 创建一个带缓冲的 channel 用于限制并发数
	semaphore := make(chan struct{}, *maxConcurrency)

	// 创建一个 WaitGroup 来等待所有 goroutine 完成
	var wg sync.WaitGroup
	totalNum := 0
	failNum := 0
	successNum := 0

	// 遍历匹配的文件
	for _, inputPath := range matches {
		// 检查是否是视频文件
		if !isVideoFile(inputPath) {
			log.Printf("跳过非视频文件: %s", inputPath)
			continue
		}

		totalNum++
		// 获取文件信息
		info, err := os.Stat(inputPath)
		if err != nil {
			failNum++
			log.Printf("无法获取文件信息: %s, 错误: %v", inputPath, err)
			continue
		}

		// 构造输出文件路径（确保输出文件是 MP4 格式）
		outputFileName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())) + ".mp4"
		outputPath := filepath.Join(*outputDir, outputFileName)

		// 检查输出路径是否与输入路径相同
		if outputPath == inputPath {
			failNum++
			log.Printf("输出路径不能与输入路径相同: %s", inputPath)
			continue
		}

		// 增加 WaitGroup 的计数器
		wg.Add(1)

		// 启动一个 goroutine 来处理视频文件
		go func(inputPath, outputPath string, fileSize int64) {
			// 限制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 在 goroutine 完成后减少 WaitGroup 的计数器
			defer wg.Done()

			// 备份原文件
			backupDir := filepath.Join(*outputDir, "backup")
			if err := os.MkdirAll(backupDir, os.ModePerm); err != nil {
				failNum++
				log.Printf("无法创建备份目录: %v", err)
				return
			}

			backupPath := filepath.Join(backupDir, filepath.Base(inputPath))
			if err := copyFile(inputPath, backupPath); err != nil {
				failNum++
				log.Printf("无法备份文件: %s, 错误: %v", inputPath, err)
				return
			}

			// 调用 ffmpeg 处理视频
			if err := processVideo(inputPath, outputPath, *watermarkPath, *watermarkWidth, *watermarkHeight, *watermarkX, *watermarkY, fileSize, *crfThreshold); err != nil {
				failNum++
				log.Printf("处理文件 %s 失败: %v", inputPath, err)
			} else {
				successNum++
				log.Printf("处理文件 %s 完成，输出到 %s", inputPath, outputPath)
			}
		}(inputPath, outputPath, info.Size())
	}

	// 等待所有 goroutine 完成
	wg.Wait() 
	log.Printf("视频处理已完成！处理数量：%v， 成功：\033[32m%v\033[0m，失败：\033[31m%v\033[0m", totalNum, successNum, failNum)
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return nil
}

// processVideo 处理视频文件（添加水印或直接转换）
func processVideo(inputPath, outputPath, watermarkPath string, width, height int, x, y string, fileSize, crfThreshold int64) error {
	// 如果没有提供水印路径，则直接转换视频
	if watermarkPath == "" {
		return convertToMP4(inputPath, outputPath, fileSize, crfThreshold)
	}

	// 否则，添加水印并转换视频
	return addWatermarkAndConvertToMP4(inputPath, outputPath, watermarkPath, width, height, x, y, fileSize, crfThreshold)
}

// convertToMP4 直接转换视频为 MP4 格式
func convertToMP4(inputPath, outputPath string, fileSize, crfThreshold int64) error {
	// 构造 ffmpeg 命令
	args := []string{
		"-i", inputPath, // 输入视频文件
		"-c:v", "libx264", // 使用 H.264 编码视频
		"-c:a", "copy",  
	}

	// 如果文件大小大于阈值，添加 -crf 28
	if fileSize > crfThreshold {
		args = append(args, "-preset", "veryslow")
		args = append(args, "-crf", "28")
	}

	// 添加输出文件路径
	args = append(args, outputPath)

	// 构造命令
	cmd := exec.Command("ffmpeg", args...)

	// 打印执行的命令
	fmt.Println("执行命令:", cmd.String())

	// 运行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg 执行失败: %v, 输出: %s", err, string(output))
	}

	return nil
}

// addWatermarkAndConvertToMP4 使用 ffmpeg 添加水印并转换为 MP4
func addWatermarkAndConvertToMP4(inputPath, outputPath, watermarkPath string, width, height int, x, y string, fileSize, crfThreshold int64) error {
	// 构造 ffmpeg 命令
	args := []string{
		"-i", inputPath,       // 输入视频文件
		"-i", watermarkPath,   // 水印图片
		"-filter_complex", fmt.Sprintf("[1:v] scale=%d:%d [wm]; [0:v][wm] overlay=%s:%s", width, height, x, y), // 动态调整水印大小和位置
		"-c:v", "libx264",   
		"-c:a", "copy",  
		"-shortest", 
	}

	// 如果文件大小大于阈值，添加 -crf 28
	if fileSize > crfThreshold {
		args = append(args, "-preset", "veryslow")
		args = append(args, "-crf", "28")
	}

	// 添加输出文件路径
	args = append(args, outputPath)

	// 构造命令
	cmd := exec.Command("ffmpeg", args...)

	// 打印执行的命令
	fmt.Println("执行命令:", cmd.String())

	// 运行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg 执行失败: %v, 输出: %s", err, string(output))
	}

	return nil
}

// isVideoFile 检查文件是否是视频文件
func isVideoFile(filename string) bool {
	// 支持的视频文件扩展名
	videoExtensions := []string{".mp4", ".mov", ".avi", ".mkv", ".flv", ".wmv"}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, videoExt := range videoExtensions {
		if ext == videoExt {
			return true
		}
	}
	return false
}
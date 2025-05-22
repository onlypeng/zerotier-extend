@echo off
REM ZerotierExtend ������ƽű�
REM ����ʽ�˵����û�ѡ�����
set EXE_PATH=.\update_planet.exe

:menu
cls
echo ==============================
echo ZerotierExtend ������Ʋ˵�
echo ==============================
echo 1. ��װ����
echo 2. ж�ط���
echo 3. ��������
echo 4. ֹͣ����
echo 5. ��������
echo 6. �˳�
echo ==============================
set /p choice=��ѡ����� (1-6): 

if "%choice%"=="1" goto install
if "%choice%"=="2" goto uninstall
if "%choice%"=="3" goto start
if "%choice%"=="4" goto stop
if "%choice%"=="5" goto restart
if "%choice%"=="6" exit /b 0

echo ��Ч��ѡ�������� 1 �� 6 ֮������֡�
pause
goto menu

:install
%EXE_PATH% install
pause
goto menu

:uninstall
%EXE_PATH% uninstall
pause
goto menu

:start
%EXE_PATH% start
pause
goto menu

:stop
%EXE_PATH% stop
pause
goto menu

:restart
%EXE_PATH% restart
pause
goto menu
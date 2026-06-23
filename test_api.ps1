$baseUrl = "http://localhost:8080/api"

Write-Host "========== Fitness Studio API Test Suite ==========" -ForegroundColor Cyan
Write-Host ""

function Invoke-APICall {
    param(
        [string]$Method,
        [string]$Endpoint,
        [hashtable]$Body = $null,
        [string]$Token = $null,
        [switch]$SkipStatusCheck
    )

    $headers = @{}
    if ($Token) {
        $headers["Authorization"] = "Bearer $Token"
    }
    $headers["Content-Type"] = "application/json"

    $url = "$baseUrl$Endpoint"

    try {
        if ($Body) {
            $jsonBody = $Body | ConvertTo-Json -Depth 10
            $response = Invoke-WebRequest -Method $Method -Uri $url -Headers $headers -Body $jsonBody -UseBasicParsing
        } else {
            $response = Invoke-WebRequest -Method $Method -Uri $url -Headers $headers -UseBasicParsing
        }

        $content = $response.Content | ConvertFrom-Json
        return [pscustomobject]@{
            StatusCode = $response.StatusCode
            Content = $content
            Success = $true
        }
    } catch {
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $errorContent = $reader.ReadToEnd()
            try {
                $errorObj = $errorContent | ConvertFrom-Json
                return [pscustomobject]@{
                    StatusCode = $_.Exception.Response.StatusCode.value__
                    Content = $errorObj
                    Success = $false
                }
            } catch {
                return [pscustomobject]@{
                    StatusCode = $_.Exception.Response.StatusCode.value__
                    Content = $errorContent
                    Success = $false
                }
            }
        }
        return [pscustomobject]@{
            StatusCode = 0
            Content = $_.Exception.Message
            Success = $false
        }
    }
}

function Write-TestResult {
    param(
        [string]$TestName,
        [bool]$Passed,
        [string]$Message = ""
    )

    if ($Passed) {
        Write-Host "[PASS] " -ForegroundColor Green -NoNewline
        Write-Host $TestName
    } else {
        Write-Host "[FAIL] " -ForegroundColor Red -NoNewline
        Write-Host $TestName
        if ($Message) {
            Write-Host "       $Message" -ForegroundColor Red
        }
    }
}

Write-Host "--- Test 1: Admin Login ---" -ForegroundColor Yellow
$adminLogin = Invoke-APICall -Method POST -Endpoint "/auth/login" -Body @{username="admin"; password="admin123"}
$adminToken = $adminLogin.Content.token
Write-TestResult -TestName "Admin login" -Passed ($adminLogin.Success -and $adminToken -ne $null) -Message $(if (!$adminLogin.Success) { $adminLogin.Content.error } )

Write-Host ""
Write-Host "--- Test 2: Member Login ---" -ForegroundColor Yellow
$memberLogin = Invoke-APICall -Method POST -Endpoint "/auth/login" -Body @{username="member1"; password="123456"}
$memberToken = $memberLogin.Content.token
Write-TestResult -TestName "Member1 login" -Passed ($memberLogin.Success -and $memberToken -ne $null) -Message $(if (!$memberLogin.Success) { $memberLogin.Content.error } )

Write-Host ""
Write-Host "--- Test 3: Invalid Login ---" -ForegroundColor Yellow
$invalidLogin = Invoke-APICall -Method POST -Endpoint "/auth/login" -Body @{username="admin"; password="wrongpass"}
Write-TestResult -TestName "Reject invalid credentials" -Passed ($invalidLogin.StatusCode -eq 401) -Message $(if ($invalidLogin.StatusCode -ne 401) { "Expected 401, got $($invalidLogin.StatusCode)" } )

Write-Host ""
Write-Host "--- Test 4: Auth Middleware (No Token) ---" -ForegroundColor Yellow
$noAuth = Invoke-APICall -Method GET -Endpoint "/members"
Write-TestResult -TestName "Reject request without token" -Passed ($noAuth.StatusCode -eq 401) -Message $(if ($noAuth.StatusCode -ne 401) { "Expected 401, got $($noAuth.StatusCode)" } )

Write-Host ""
Write-Host "--- Test 5: Auth Middleware (Member Token for Admin Endpoint) ---" -ForegroundColor Yellow
$memberForbidden = Invoke-APICall -Method GET -Endpoint "/members" -Token $memberToken
Write-TestResult -TestName "Reject member accessing admin endpoint" -Passed ($memberForbidden.StatusCode -eq 403) -Message $(if ($memberForbidden.StatusCode -ne 403) { "Expected 403, got $($memberForbidden.StatusCode)" } )

Write-Host ""
Write-Host "--- Test 6: Get Current User Info ---" -ForegroundColor Yellow
$me = Invoke-APICall -Method GET -Endpoint "/auth/me" -Token $memberToken
Write-TestResult -TestName "Get current user info" -Passed ($me.Success -and $me.Content.user.name -eq "王小明") -Message $(if (!$me.Success) { $me.Content.error } )

Write-Host ""
Write-Host "--- Test 7: Get Member List (Admin) ---" -ForegroundColor Yellow
$memberList = Invoke-APICall -Method GET -Endpoint "/members?page=1&page_size=10" -Token $adminToken
Write-TestResult -TestName "Admin get member list" -Passed ($memberList.Success -and $memberList.Content.members.Count -ge 2) -Message $(if (!$memberList.Success) { $memberList.Content.error } )

Write-Host ""
Write-Host "--- Test 8: Get Coaches ---" -ForegroundColor Yellow
$coaches = Invoke-APICall -Method GET -Endpoint "/coaches" -Token $memberToken
Write-TestResult -TestName "Get coach list" -Passed ($coaches.Success -and $coaches.Content.coaches.Count -ge 3) -Message $(if (!$coaches.Success) { $coaches.Content.error } )

Write-Host ""
Write-Host "--- Test 9: Get Courses ---" -ForegroundColor Yellow
$courses = Invoke-APICall -Method GET -Endpoint "/courses" -Token $memberToken
Write-TestResult -TestName "Get course list" -Passed ($courses.Success -and $courses.Content.courses.Count -ge 4) -Message $(if (!$courses.Success) { $courses.Content.error } )

Write-Host ""
Write-Host "--- Test 10: Get Schedules ---" -ForegroundColor Yellow
$schedules = Invoke-APICall -Method GET -Endpoint "/schedules" -Token $memberToken
Write-TestResult -TestName "Get schedule list" -Passed ($schedules.Success -and $schedules.Content.schedules.Count -ge 4) -Message $(if (!$schedules.Success) { $schedules.Content.error } )
$scheduleId = $schedules.Content.schedules[0].id
Write-Host "       Using schedule ID: $scheduleId" -ForegroundColor Gray

Write-Host ""
Write-Host "--- Test 11: Create Booking ---" -ForegroundColor Yellow
$booking = Invoke-APICall -Method POST -Endpoint "/bookings" -Token $memberToken -Body @{schedule_id = [int]$scheduleId}
Write-TestResult -TestName "Create booking" -Passed ($booking.Success) -Message $(if (!$booking.Success) { $booking.Content.error } )
$bookingId = $booking.Content.booking.id
Write-Host "       Booking ID: $bookingId" -ForegroundColor Gray

Write-Host ""
Write-Host "--- Test 12: Duplicate Booking ---" -ForegroundColor Yellow
$dupBooking = Invoke-APICall -Method POST -Endpoint "/bookings" -Token $memberToken -Body @{schedule_id = [int]$scheduleId}
Write-TestResult -TestName "Prevent duplicate booking" -Passed ($dupBooking.StatusCode -eq 400 -and $dupBooking.Content.error -like "*重复*") -Message $(if ($dupBooking.StatusCode -ne 400) { "Expected 400, got $($dupBooking.StatusCode): $($dupBooking.Content.error)" } )

Write-Host ""
Write-Host "--- Test 13: Get My Bookings ---" -ForegroundColor Yellow
$myBookings = Invoke-APICall -Method GET -Endpoint "/bookings/my" -Token $memberToken
Write-TestResult -TestName "Get my bookings" -Passed ($myBookings.Success -and $myBookings.Content.bookings.Count -ge 1) -Message $(if (!$myBookings.Success) { $myBookings.Content.error } )

Write-Host ""
Write-Host "--- Test 14: Cancel Booking ---" -ForegroundColor Yellow
$cancel = Invoke-APICall -Method PUT -Endpoint "/bookings/$bookingId/cancel" -Token $memberToken
Write-TestResult -TestName "Cancel booking" -Passed ($cancel.Success -and $cancel.Content.booking.status -eq "canceled") -Message $(if (!$cancel.Success) { $cancel.Content.error } )

Write-Host ""
Write-Host "--- Test 15: Re-book After Cancel ---" -ForegroundColor Yellow
$reBooking = Invoke-APICall -Method POST -Endpoint "/bookings" -Token $memberToken -Body @{schedule_id = [int]$scheduleId}
Write-TestResult -TestName "Re-book after cancel" -Passed ($reBooking.Success) -Message $(if (!$reBooking.Success) { $reBooking.Content.error } )
$bookingId2 = $reBooking.Content.booking.id

Write-Host ""
Write-Host "--- Test 16: Check-in (Admin) ---" -ForegroundColor Yellow
$checkin = Invoke-APICall -Method PUT -Endpoint "/bookings/$bookingId2/checkin" -Token $adminToken
Write-TestResult -TestName "Admin check-in member" -Passed ($checkin.Success -and $checkin.Content.booking.status -eq "checked") -Message $(if (!$checkin.Success) { $checkin.Content.error } )

Write-Host ""
Write-Host "--- Test 17: Renew Membership ---" -ForegroundColor Yellow
$renew = Invoke-APICall -Method POST -Endpoint "/members/renew" -Token $memberToken -Body @{months = 3}
Write-TestResult -TestName "Renew membership" -Passed ($renew.Success -and $renew.Content.months -eq 3) -Message $(if (!$renew.Success) { $renew.Content.error } )

Write-Host ""
Write-Host "--- Test 18: Get Expiring Soon Members ---" -ForegroundColor Yellow
$expiring = Invoke-APICall -Method GET -Endpoint "/members/expiring-soon?days=30" -Token $adminToken
Write-TestResult -TestName "Get expiring members" -Passed ($expiring.Success) -Message $(if (!$expiring.Success) { $expiring.Content.error } )

Write-Host ""
Write-Host "--- Test 19: Admin Creates Coach ---" -ForegroundColor Yellow
$newCoach = Invoke-APICall -Method POST -Endpoint "/coaches" -Token $adminToken -Body @{name="测试教练"; phone="13800000099"; specialty="综合训练"}
Write-TestResult -TestName "Admin creates coach" -Passed ($newCoach.Success) -Message $(if (!$newCoach.Success) { $newCoach.Content.error } )

Write-Host ""
Write-Host "--- Test 20: Admin Creates Course ---" -ForegroundColor Yellow
$newCourse = Invoke-APICall -Method POST -Endpoint "/courses" -Token $adminToken -Body @{name="测试课程"; description="测试"; duration=30}
Write-TestResult -TestName "Admin creates course" -Passed ($newCourse.Success) -Message $(if (!$newCourse.Success) { $newCourse.Content.error } )

Write-Host ""
Write-Host "--- Test 21: Get Dashboard Stats ---" -ForegroundColor Yellow
$dashboard = Invoke-APICall -Method GET -Endpoint "/stats/dashboard" -Token $adminToken
Write-TestResult -TestName "Get dashboard stats" -Passed ($dashboard.Success -and $dashboard.Content.members -ne $null) -Message $(if (!$dashboard.Success) { $dashboard.Content.error } )

Write-Host ""
Write-Host "--- Test 22: Get Weekly Attendance ---" -ForegroundColor Yellow
$attendance = Invoke-APICall -Method GET -Endpoint "/stats/weekly-attendance" -Token $adminToken
Write-TestResult -TestName "Get weekly attendance" -Passed ($attendance.Success) -Message $(if (!$attendance.Success) { $attendance.Content.error } )

Write-Host ""
Write-Host "--- Test 23: Get Member Activity ---" -ForegroundColor Yellow
$activity = Invoke-APICall -Method GET -Endpoint "/stats/member-activity" -Token $adminToken
Write-TestResult -TestName "Get member activity" -Passed ($activity.Success) -Message $(if (!$activity.Success) { $activity.Content.error } )

Write-Host ""
Write-Host "--- Test 24: Full Capacity Test ---" -ForegroundColor Yellow
$smallSchedule = Invoke-APICall -Method POST -Endpoint "/schedules" -Token $adminToken -Body @{
    course_id = 1
    coach_id = 1
    start_time = (Get-Date).AddDays(1).ToString("yyyy-MM-ddTHH:mm:ss")
    end_time = (Get-Date).AddDays(1).AddHours(1).ToString("yyyy-MM-ddTHH:mm:ss")
    capacity = 1
    room = "测试房"
}
if ($smallSchedule.Success) {
    $smallScheduleId = $smallSchedule.Content.schedule.id
    
    $member2Login = Invoke-APICall -Method POST -Endpoint "/auth/login" -Body @{username="member2"; password="123456"}
    $member2Token = $member2Login.Content.token
    
    $booking1 = Invoke-APICall -Method POST -Endpoint "/bookings" -Token $memberToken -Body @{schedule_id = [int]$smallScheduleId}
    $booking2 = Invoke-APICall -Method POST -Endpoint "/bookings" -Token $member2Token -Body @{schedule_id = [int]$smallScheduleId}
    
    Write-TestResult -TestName "Prevent booking when full" -Passed ($booking2.StatusCode -eq 400 -and $booking2.Content.error -like "*已满*") -Message $(if ($booking2.StatusCode -ne 400) { "Expected 400, got $($booking2.StatusCode): $($booking2.Content.error)" } )
} else {
    Write-TestResult -TestName "Prevent booking when full" -Passed $false -Message "Failed to create test schedule: $($smallSchedule.Content.error)"
}

Write-Host ""
Write-Host "--- Test 25: User Registration ---" -ForegroundColor Yellow
$randomUser = "testuser_" + (Get-Random -Minimum 1000 -Maximum 9999)
$register = Invoke-APICall -Method POST -Endpoint "/auth/register" -Body @{
    username = $randomUser
    password = "test123"
    name = "测试用户"
    phone = "13900001111"
}
Write-TestResult -TestName "User registration" -Passed ($register.Success -and $register.Content.token -ne $null) -Message $(if (!$register.Success) { $register.Content.error } )

Write-Host ""
Write-Host "========== Test Summary ==========" -ForegroundColor Cyan
Write-Host "API Server: $baseUrl" -ForegroundColor Gray
Write-Host "Admin Account: admin / admin123" -ForegroundColor Gray
Write-Host "Member Account: member1 / 123456" -ForegroundColor Gray
Write-Host "Member Account: member2 / 123456" -ForegroundColor Gray
Write-Host ""
Write-Host "All API tests completed!" -ForegroundColor Green

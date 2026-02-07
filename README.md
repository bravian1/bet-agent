# ⚽ AI Betting Agent

An automated betting recommendation system powered by Google Gemini AI. The agent discovers daily football matches, analyzes each game with deep research, and sends personalized betting slips via email.

## 🎯 Features

- **Automated Discovery**: Finds today's football matches from major leagues
- **Deep AI Analysis**: Each game analyzed with Gemini's thinking mode
- **Smart Research**: Uses Google Search + URL Context for comprehensive data
- **Email Delivery**: HTML-formatted betting slips sent to your inbox
- **Performance Optimization**: Daily analysis of results with improvement suggestions
- **Configurable Prompts**: Easy-to-edit prompt files for fine-tuning analysis
- **Rate Limiting**: Controlled API usage to stay within limits
- **Scheduled Execution**: Automatic cron-based workflows

## 📋 Requirements

- Go 1.21 or higher
- Google Gemini API key ([Get one here](https://aistudio.google.com/app/apikey))
- SMTP email account (Gmail recommended)

## 🚀 Quick Start

### 1. Clone and Setup

```bash
# Create project directory
mkdir bet-agent
cd bet-agent

# Copy the main code
# (paste bet-agent-complete.go as main.go)

# Initialize Go module
go mod init bet-agent
go mod tidy
```

### 2. Configure Environment

```bash
# Copy the example environment file
cp .env.example .env

# Edit with your credentials
nano .env  # or use your preferred editor
```

**Required Configuration:**

```env
# Get your API key from: https://aistudio.google.com/app/apikey
GEMINI_API_KEY=your_gemini_api_key_here

# For Gmail, use App Password (not regular password)
# Generate at: https://myaccount.google.com/apppasswords
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASSWORD=your_app_password_here
EMAIL_FROM=your_email@gmail.com
EMAIL_TO=recipient@example.com

# Cron schedules
MAIN_CRON=0 9 * * *         # 9 AM daily
OPTIMIZATION_CRON=0 22 * * * # 10 PM daily
```

### 3. Setup Gmail App Password (If using Gmail)

1. Go to [Google Account Security](https://myaccount.google.com/security)
2. Enable 2-Step Verification
3. Go to [App Passwords](https://myaccount.google.com/apppasswords)
4. Generate a new app password for "Mail"
5. Use this password in `SMTP_PASSWORD`

### 4. Run the Agent

```bash
# Build and run
go run main.go

# Or build executable
go build -o bet-agent main.go
./bet-agent
```

## 📁 Project Structure

```
bet-agent/
├── main.go              # Main application code
├── .env                 # Your configuration (DO NOT COMMIT)
├── .env.example         # Configuration template
├── go.mod               # Go dependencies
├── go.sum               # Dependency checksums
├── data/                # Generated data (auto-created)
│   ├── YYYY-MM-DD_games.json
│   ├── YYYY-MM-DD_slip.json
│   └── YYYY-MM-DD_optimization.json
└── prompts/             # AI prompts (auto-created, EDIT THESE!)
    ├── discovery.txt
    ├── analysis.txt
    ├── analysis_system.txt
    ├── optimization.txt
    └── optimization_system.txt
```

## 🎨 Customizing Analysis

The agent's behavior is controlled by prompt files in the `prompts/` directory:

### `prompts/discovery.txt`
Controls which leagues and competitions to search for.

**Example customization:**
```
Search for fixtures from these leagues:
- English Premier League
- La Liga (Spain)
- NBA (Basketball)  # Add other sports!
```

### `prompts/analysis.txt`
Controls how games are analyzed and what factors to consider.

**Example customization:**
```
RESEARCH CHECKLIST:
1. Recent Form: Last 5 matches
2. Head-to-Head: Last 5 meetings
3. Weather conditions  # Add new factors!
4. Referee statistics  # Add new factors!
```

### `prompts/analysis_system.txt`
Controls the AI's personality and approach.

**Example customization:**
```
You are a conservative analyst.
Only recommend bets with 70%+ confidence.
Prioritize value over volume.
```

## 🔧 Configuration Options

### Cron Schedule Format

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6)
│ │ │ │ │
* * * * *
```

**Examples:**
- `0 9 * * *` - Every day at 9 AM
- `0 8 * * 1-5` - Weekdays at 8 AM
- `*/30 * * * *` - Every 30 minutes
- `0 0 * * 0` - Sundays at midnight

### Rate Limiting

Adjust `RATE_LIMIT_SECONDS` to control delay between API calls:
- `3` = 3 seconds (default)
- `5` = 5 seconds (safer for rate limits)
- `1` = 1 second (faster but risky)

### Model Selection

You can change which Gemini models to use:

```env
DISCOVERY_MODEL=gemini-2.0-flash-exp          # Fast searches
ANALYSIS_MODEL=gemini-2.0-flash-thinking-exp  # Deep thinking
OPTIMIZATION_MODEL=gemini-2.0-flash-exp       # Cost-effective
```

Available models:
- `gemini-2.0-flash-exp` - Fast, efficient
- `gemini-2.0-flash-thinking-exp` - Deep reasoning
- `gemini-pro` - Balanced

## 📊 How It Works

### Main Workflow (Daily)

1. **Discovery** 🔍
   - Searches for today's football matches
   - Saves to `data/YYYY-MM-DD_games.json`

2. **Analysis** 🔬
   - For each game:
     - Researches form, stats, news
     - Uses Gemini thinking mode
     - Generates recommendation
     - Rate limits between calls

3. **Compilation** 📋
   - Creates betting slip
   - Calculates accumulator odds
   - Saves to `data/YYYY-MM-DD_slip.json`

4. **Distribution** 📧
   - Formats HTML email
   - Sends to configured address

### Optimization Workflow (Nightly)

1. **Results Check** ✅
   - Loads yesterday's slip
   - Searches for match results

2. **Analysis** 📊
   - Calculates win/loss
   - Identifies patterns
   - Suggests improvements

3. **Reporting** 📧
   - Generates performance report
   - Sends optimization email

## 🎯 Usage Examples

### Run Once (Test Mode)

```bash
# Run immediately without waiting for cron
go run main.go
```

The agent runs the main workflow immediately on startup, then waits for scheduled times.

### Run in Background

```bash
# Using nohup
nohup go run main.go > bet-agent.log 2>&1 &

# Using screen
screen -S bet-agent
go run main.go
# Ctrl+A, D to detach

# Using systemd (see below)
```

### Check Logs

```bash
# If using nohup
tail -f bet-agent.log

# If using screen
screen -r bet-agent
```

## 🐳 Docker Deployment (Optional)

Create `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o bet-agent main.go

CMD ["./bet-agent"]
```

Build and run:

```bash
docker build -t bet-agent .
docker run -d --env-file .env bet-agent
```

## 🔒 Security Best Practices

1. **Never commit `.env`** - Add to `.gitignore`
2. **Use app passwords** - Never use main email password
3. **Rotate API keys** - Change periodically
4. **Secure email** - Use dedicated email for automation
5. **Monitor logs** - Check for suspicious activity

## 🐛 Troubleshooting

### "GEMINI_API_KEY is required"

**Solution:** Set your API key in `.env` file

```env
GEMINI_API_KEY=your_actual_key_here
```

### "Authentication failed" (Email)

**Solution:** Use Gmail App Password, not regular password
1. Enable 2FA on Google Account
2. Generate App Password
3. Use App Password in `SMTP_PASSWORD`

### "No games found for today"

**Possible causes:**
- No matches scheduled today
- Search queries need adjustment
- Rate limiting from search engines

**Solution:** Edit `prompts/discovery.txt` to search different leagues

### "Failed to parse JSON"

**Possible causes:**
- Gemini returned markdown instead of pure JSON
- Invalid JSON format

**Solution:** 
- Check `ResponseMIMEType` is set to `"application/json"`
- Review system prompts to emphasize JSON-only output

### Rate Limiting Errors

**Solution:** Increase `RATE_LIMIT_SECONDS` in `.env`

```env
RATE_LIMIT_SECONDS=5  # or higher
```

## 📈 Improving Performance

### 1. Refine Prompts

Monitor the `optimization` reports and adjust prompts based on suggestions:

```bash
# Check yesterday's optimization
cat data/YYYY-MM-DD_optimization.json
```

### 2. Focus on High-Confidence Bets

Edit `prompts/analysis_system.txt`:

```
Only recommend bets with High confidence.
Skip any bet below 70% certainty.
```

### 3. Track Historical Performance

```bash
# View all optimization reports
ls -la data/*_optimization.json

# Analyze patterns
jq '.success_rate' data/*_optimization.json
```

### 4. Add More Data Sources

Edit `prompts/analysis.txt` to include specific data sources:

```
Research these sites:
- WhoScored.com for detailed stats
- Understat.com for xG data
- Football-Data.co.uk for historical data
```

## 🔄 Systemd Service (Linux)

Create `/etc/systemd/system/bet-agent.service`:

```ini
[Unit]
Description=AI Betting Agent
After=network.target

[Service]
Type=simple
User=your_user
WorkingDirectory=/path/to/bet-agent
ExecStart=/usr/local/go/bin/go run main.go
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable bet-agent
sudo systemctl start bet-agent
sudo systemctl status bet-agent
```

## 📝 Logging

The agent logs all activity to stdout. Capture logs:

```bash
# Redirect to file
go run main.go >> bet-agent.log 2>&1

# Or use systemd journalctl
sudo journalctl -u bet-agent -f
```

## ⚠️ Disclaimer

This tool is for **educational and entertainment purposes only**. 

- Betting carries financial risk
- AI predictions are not guaranteed
- Past performance doesn't predict future results
- Gamble responsibly
- Know your local gambling laws
- Never bet more than you can afford to lose

## 🤝 Contributing

Ideas for improvement:
- [ ] Add more sports (basketball, tennis, etc.)
- [ ] Support multiple betting sites
- [ ] Add Telegram notifications
- [ ] Create web dashboard
- [ ] Add bankroll management
- [ ] Implement Kelly Criterion
- [ ] Add live betting support
- [ ] Multi-language support

## 📄 License

MIT License - Feel free to modify and use as you wish!

## 🆘 Support

Having issues? 

1. Check the troubleshooting section above
2. Review your `.env` configuration
3. Check `data/` folder for error logs in JSON files
4. Verify your Gemini API key is valid
5. Test SMTP settings with a simple email script first

## 🎓 How to Improve the Agent

### Week 1: Run and Observe
- Let it run for a week
- Don't change anything
- Collect baseline data

### Week 2: Analyze Patterns
- Review optimization reports
- Identify which markets work best
- Note common failure patterns

### Week 3: Refine Prompts
- Update `analysis.txt` based on insights
- Add emphasis to winning factors
- Remove focus from losing factors

### Week 4: Test and Iterate
- A/B test prompt changes
- Compare results week-over-week
- Iterate on successful changes

**Remember:** The prompts are the key to improving performance. Small changes can have big impacts!

---

Made with 🤖 AI + ⚽ Football + 💡 Innovation

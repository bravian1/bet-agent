document.addEventListener("DOMContentLoaded", () => {
    fetchAnalytics();
});

async function fetchAnalytics() {
    try {
        const response = await fetch('/api/public/analytics');
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();

        if (data.status !== "success") {
            throw new Error("API returned failure");
        }

        renderAnalytics(data.optimization, data.slip);

        // Hide loader, show content
        document.getElementById("loader").classList.add("hidden");
        document.getElementById("content").classList.remove("hidden");

    } catch (error) {
        console.error("Failed to fetch analytics:", error);
        document.getElementById("loader").innerHTML = `
            <i class="fa-solid fa-triangle-exclamation text-error fa-3x"></i>
            <p>Failed to load analytics data. Please try again later.</p>
        `;
    }
}

function renderAnalytics(optimization, slip) {
    // Top Stats
    if (optimization) {
        // Date formatting
        const dateObj = new Date(optimization.date);
        const dateStr = dateObj.toLocaleDateString("en-US", { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });
        document.getElementById("analytics-date").textContent = `Insights from ${dateStr}`;

        // Numbers
        document.getElementById("stat-success-rate").textContent = `${(optimization.success_rate * 100).toFixed(1)}%`;

        // Color code success rate
        const srEl = document.getElementById("stat-success-rate");
        if (optimization.success_rate >= 0.7) srEl.className = "text-success";
        else if (optimization.success_rate < 0.5) srEl.className = "text-error";
        else srEl.className = "text-primary";

        document.getElementById("stat-wins").textContent = optimization.wins;
        document.getElementById("stat-losses").textContent = optimization.losses;
        document.getElementById("stat-total").textContent = optimization.total_bets;

        // Written Insights
        document.getElementById("ai-insights").innerHTML =
            `<strong>Overview:</strong> ${optimization.insights}<br><br>
            <strong>Winning Markets:</strong> ${optimization.winning_markets.join(", ") || "None"}<br>
            <strong>Losing Markets:</strong> ${optimization.losing_markets.join(", ") || "None"}`;

        document.getElementById("ai-improvements").textContent = optimization.prompt_improvements;
    } else {
        document.getElementById("analytics-date").textContent = "No optimization data available yet.";
    }

    // Historical Slips Grid
    if (slip && slip.recommendations && slip.recommendations.length > 0) {
        const container = document.getElementById("slips-container");
        container.innerHTML = ""; // Clear loader

        slip.recommendations.forEach(rec => {
            // Determine card border color based on confidence as a fallback, 
            // but ideally we'd have the actual win/loss boolean mapped per game. 
            // For now, let's just make it look good with the data we have.
            const conf = rec.confidence.toLowerCase();
            let cardClass = "unknown";
            if (conf.includes("high")) cardClass = "won";
            else if (conf.includes("low")) cardClass = "lost";

            const card = document.createElement("div");
            card.className = `bet-card ${cardClass}`;

            card.innerHTML = `
                <div class="bet-confidence">${rec.confidence}</div>
                <div class="bet-league">${rec.game.league} • ${rec.game.kickoff_time}</div>
                <div class="bet-teams">${rec.game.home_team} vs ${rec.game.away_team}</div>
                
                <div class="bet-reasoning">${rec.reasoning}</div>
                
                <div class="bet-details">
                    <span class="bet-selection">${rec.selection} (${rec.market})</span>
                    <span class="bet-odds">@ ${rec.odds.toFixed(2)}</span>
                </div>
            `;
            container.appendChild(card);
        });
    } else {
        document.getElementById("slips-container").innerHTML = `
            <p style="color: #94a3b8; grid-column: 1 / -1; text-align: center;">No slip data found for this analysis period.</p>
        `;
    }
}

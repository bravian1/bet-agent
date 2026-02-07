document.getElementById('subscribeForm').addEventListener('submit', async function (e) {
    e.preventDefault();

    const emailInput = document.getElementById('email');
    const submitBtn = document.getElementById('submitBtn');
    const btnText = submitBtn.querySelector('.btn-text');
    const messageDiv = document.getElementById('message');

    // Reset state
    messageDiv.className = 'message';
    messageDiv.textContent = '';

    // Loading state
    const originalText = btnText.textContent;
    btnText.textContent = 'Joining...';
    submitBtn.disabled = true;

    try {
        const response = await fetch('/api/subscribe', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ email: emailInput.value }),
        });

        const data = await response.json();

        messageDiv.textContent = data.message;
        messageDiv.classList.add('show');

        if (data.status === 'success') {
            messageDiv.classList.add('success');
            emailInput.value = '';
            btnText.textContent = 'Joined!';

            // Revert button after delay
            setTimeout(() => {
                btnText.textContent = originalText;
                submitBtn.disabled = false;
            }, 3000);

        } else if (data.status === 'exists') {
            messageDiv.classList.add('exists');
            submitBtn.disabled = false;
            btnText.textContent = originalText;
        } else {
            throw new Error('Unknown error');
        }

    } catch (error) {
        console.error('Error:', error);
        messageDiv.textContent = 'Something went wrong. Please try again.';
        messageDiv.classList.add('show', 'error');
        submitBtn.disabled = false;
        btnText.textContent = originalText;
    }
});

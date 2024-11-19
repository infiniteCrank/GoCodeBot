const connection = new WebSocket('ws://localhost:8080/ws');

connection.onopen = () => {
    console.log('WebSocket connected');
};

connection.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    
    if (msg.type === "response") {
        const messagesContainer = document.getElementById('messages');
        messagesContainer.innerHTML += `<div>${msg.response}</div>`;
        
        // Show feedback options after displaying the response
        showFeedbackOptions(msg.response);
    }
};

// Show the feedback options after receiving a response
function showFeedbackOptions(response) {
    const feedbackDiv = document.getElementById('feedback');
    feedbackDiv.style.display = "block"; // Show feedback options
    
    // Store the last response and query for feedback submission
    window.lastQuery = document.getElementById('query').value;
    window.lastResponse = response;
}

// Function to submit feedback
function submitFeedback(rating) {
    connection.send(JSON.stringify({ 
        type: "feedback", 
        query: window.lastQuery, 
        response: window.lastResponse, 
        rating: rating 
    }));

    // Hide feedback options after submission
    const feedbackDiv = document.getElementById('feedback');
    feedbackDiv.style.display = "none";
    
    // Reset the input field after sending feedback
    document.getElementById('query').value = ''; 
}

// Event listener for the send button
document.getElementById('send').onclick = () => {
    const query = document.getElementById('query').value;
    connection.send(JSON.stringify({ type: "query", query }));
    document.getElementById('query').value = ''; // Clear input field
};

document.getElementById('send').onclick = () => {
    const query = document.getElementById('query').value;
    connection.send(JSON.stringify({ query }));
    document.getElementById('query').value = ''; // Clear input field
};

document.getElementById('send').onclick = () => {
    const query = document.getElementById('query').value;
    connection.send(JSON.stringify({ type: "query", query }));
    document.getElementById('query').value = ''; // Clear input field
};

// Feedback submission
function submitFeedback(query, response, rating) {
    connection.send(JSON.stringify({ 
        type: "feedback", 
        query: query, 
        response: response, 
        rating: rating 
    }));
}

// Example of using the feedback
// Call submitFeedback after a response is displayed to the user
// You can implement a simple rating UI using buttons for 1 to 5 stars
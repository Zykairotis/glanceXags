FROM python:3.11-slim

# Install UV
RUN pip install uv

# Set working directory
WORKDIR /app

# Copy project files
COPY . ./

# Install dependencies using UV
RUN uv pip install --system -e .

# Expose the port
EXPOSE 9200

# Run the application
CMD ["python", "data_api.py"]
